package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var (
	dockerImageListPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:@-]{0,254}$`)
)

func dockerRandomSuffix() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// RegistryRepositories 获取镜像市场镜像列表（分页，内置清单）。
func (s *DockerOperationService) RegistryRepositories(query string, page, pageSize int) ([]map[string]any, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	q := strings.ToLower(strings.TrimSpace(query))
	filtered := make([]marketImage, 0, len(marketImages))
	for _, m := range marketImages {
		if q != "" &&
			!strings.Contains(strings.ToLower(m.Name), q) &&
			!strings.Contains(strings.ToLower(m.Description), q) &&
			!strings.Contains(strings.ToLower(m.Category), q) {
			continue
		}
		filtered = append(filtered, m)
	}
	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	items := make([]map[string]any, 0, end-start)
	for _, m := range filtered[start:end] {
		items = append(items, map[string]any{
			"name":        m.Name,
			"description": m.Description,
			"category":    m.Category,
			"starCount":   m.StarCount,
			"pullRef":     m.Name,
		})
	}
	return items, total, nil
}

// ImagePull 拉取镜像，支持 Docker Hub 与国内镜像站前缀。
func (s *DockerOperationService) ImagePull(ctx context.Context, assetType string, assetID, connectionID uint, image string) (string, error) {
	image = strings.TrimSpace(image)
	if !dockerImagePattern.MatchString(image) {
		return "", errors.New("镜像名称无效")
	}
	out, err := s.runLong(ctx, assetType, assetID, connectionID, "docker pull "+dockerShellQuote(image), 8<<20)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func validateImageList(images []string) error {
	if len(images) == 0 || len(images) > 50 {
		return errors.New("请选择 1–50 个镜像")
	}
	for _, image := range images {
		image = strings.TrimSpace(image)
		if !dockerImageListPattern.MatchString(image) {
			return errors.New("镜像名称无效：" + image)
		}
	}
	return nil
}

func joinImageArgs(images []string) string {
	quoted := make([]string, 0, len(images))
	for _, image := range images {
		quoted = append(quoted, dockerShellQuote(strings.TrimSpace(image)))
	}
	return strings.Join(quoted, " ")
}

// ImageRemove 批量删除镜像。
func (s *DockerOperationService) ImageRemove(ctx context.Context, assetType string, assetID, connectionID uint, images []string, force bool) (string, error) {
	if err := validateImageList(images); err != nil {
		return "", err
	}
	command := "docker rmi " + joinImageArgs(images)
	if force {
		command += " --force"
	}
	out, err := s.run(ctx, assetType, assetID, connectionID, command, 4<<20)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ImageTag 为镜像打新标签（复制）。
func (s *DockerOperationService) ImageTag(ctx context.Context, assetType string, assetID, connectionID uint, source, target string) (string, error) {
	source, target = strings.TrimSpace(source), strings.TrimSpace(target)
	if !dockerImagePattern.MatchString(source) || !dockerImagePattern.MatchString(target) {
		return "", errors.New("镜像名称无效")
	}
	out, err := s.run(ctx, assetType, assetID, connectionID, "docker tag "+dockerShellQuote(source)+" "+dockerShellQuote(target), 64<<10)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ImageRename 重命名镜像（打新标签并删除旧标签）。
func (s *DockerOperationService) ImageRename(ctx context.Context, assetType string, assetID, connectionID uint, source, target string) (string, error) {
	source, target = strings.TrimSpace(source), strings.TrimSpace(target)
	if !dockerImagePattern.MatchString(source) || !dockerImagePattern.MatchString(target) {
		return "", errors.New("镜像名称无效")
	}
	if source == target {
		return "", errors.New("新旧名称相同")
	}
	if _, err := s.run(ctx, assetType, assetID, connectionID, "docker tag "+dockerShellQuote(source)+" "+dockerShellQuote(target), 64<<10); err != nil {
		return "", err
	}
	// 仅当源为名称标签时删除旧标签，避免误删共享镜像。
	out, err := s.run(ctx, assetType, assetID, connectionID, "docker rmi "+dockerShellQuote(source), 64<<10)
	if err != nil {
		return "", errors.New("新标签已创建，但删除旧标签失败：" + err.Error())
	}
	return strings.TrimSpace(out), nil
}

// ImageTransfer 将镜像导出并导入到另一台机器（平台主机/物理机资产）的 Docker。
// migrate=true 时会在成功导入后删除源机器上的镜像（迁移）；
// migrate=false 时保留源镜像（复制）。
func (s *DockerOperationService) ImageTransfer(ctx context.Context, assetType string, assetID, connectionID uint, image, targetAssetType string, targetAssetID uint, migrate bool) (string, error) {
	image = strings.TrimSpace(image)
	if !dockerImagePattern.MatchString(image) {
		return "", errors.New("镜像名称无效")
	}
	if targetAssetType != "host" && targetAssetType != "physical" {
		return "", errors.New("Docker 目标仅支持主机或物理机资产")
	}

	// 1. 源机器导出镜像到临时 tar。
	srcRemote := "/tmp/autoops-docker-" + dockerRandomSuffix() + ".tar"
	if _, err := s.runLong(ctx, assetType, assetID, connectionID, "docker save -o "+srcRemote+" "+dockerShellQuote(image), 64<<20); err != nil {
		return "", errors.New("源主机导出镜像失败：" + err.Error())
	}

	// 2. 下载到后端本地临时文件（流式，避免占用内存）。
	local, err := os.CreateTemp("", "autoops-docker-transfer-*.tar")
	if err != nil {
		return "", errors.New("创建本地临时文件失败")
	}
	localPath := local.Name()
	defer os.Remove(localPath)
	sc, client, err := s.sftpClient(ctx, assetType, assetID, connectionID)
	if err != nil {
		local.Close()
		return "", err
	}
	srcFile, err := sc.Open(srcRemote)
	if err != nil {
		sc.Close()
		client.Close()
		local.Close()
		return "", errors.New("读取源主机镜像文件失败")
	}
	if _, err := io.Copy(local, srcFile); err != nil {
		srcFile.Close()
		sc.Close()
		client.Close()
		local.Close()
		return "", errors.New("下载镜像文件失败")
	}
	srcFile.Close()
	local.Close()
	_ = sc.Remove(srcRemote)
	sc.Close()
	client.Close()

	// 3. 通过平台连接档案连接目标机器并上传 tar。
	tClient, err := ResolveAssetSSH(s.db, targetAssetType, targetAssetID, 0)
	if err != nil {
		return "", errors.New("目标主机 SSH 连接失败：" + err.Error())
	}
	defer tClient.Close()
	tSftp, err := sftp.NewClient(tClient)
	if err != nil {
		return "", errors.New("目标主机 SFTP 初始化失败")
	}
	defer tSftp.Close()

	dstRemote := "/tmp/autoops-docker-" + dockerRandomSuffix() + ".tar"
	dstFile, err := tSftp.Create(dstRemote)
	if err != nil {
		return "", errors.New("目标主机创建临时文件失败")
	}
	lf, err := os.Open(localPath)
	if err != nil {
		dstFile.Close()
		return "", errors.New("读取本地镜像文件失败")
	}
	if _, err := io.Copy(dstFile, lf); err != nil {
		lf.Close()
		dstFile.Close()
		return "", errors.New("上传镜像文件失败")
	}
	lf.Close()
	dstFile.Close()
	defer tSftp.Remove(dstRemote)

	// 4. 目标机器导入镜像。
	out, err := runDockerSSH(ctx, tClient, "docker load -i "+dstRemote, 8<<20, 15*time.Minute)
	if err != nil {
		return "", errors.New("目标主机导入镜像失败：" + err.Error())
	}

	// 5. 迁移模式下删除源机器镜像。
	if migrate {
		if _, err := s.run(ctx, assetType, assetID, connectionID, "docker rmi "+dockerShellQuote(image), 64<<10); err != nil {
			return strings.TrimSpace(out), errors.New("镜像已复制到目标，但删除源镜像失败：" + err.Error())
		}
	}
	return strings.TrimSpace(out), nil
}

// sftpClient 在资产上建立 SFTP 连接。
func (s *DockerOperationService) sftpClient(ctx context.Context, assetType string, assetID, connectionID uint) (*sftp.Client, *ssh.Client, error) {
	client, err := ResolveAssetSSH(s.db, assetType, assetID, connectionID)
	if err != nil {
		return nil, nil, errors.New("SSH 连接失败，请检查连接档案、凭据和主机指纹")
	}
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stop()
	sc, err := sftp.NewClient(client)
	if err != nil {
		_ = client.Close()
		return nil, nil, errors.New("无法建立 SFTP 文件通道")
	}
	return sc, client, nil
}

// ImageExport 将多个镜像导出为 tar 包字节流。
func (s *DockerOperationService) ImageExport(ctx context.Context, assetType string, assetID, connectionID uint, images []string) ([]byte, string, error) {
	if err := validateImageList(images); err != nil {
		return nil, "", err
	}
	remote := "/tmp/autoops-docker-" + dockerRandomSuffix() + ".tar"
	if _, err := s.runLong(ctx, assetType, assetID, connectionID, "docker save -o "+remote+" "+joinImageArgs(images), 64<<20); err != nil {
		return nil, "", err
	}
	sc, client, err := s.sftpClient(ctx, assetType, assetID, connectionID)
	if err != nil {
		return nil, "", err
	}
	defer sc.Close()
	defer client.Close()
	f, err := sc.Open(remote)
	if err != nil {
		return nil, "", errors.New("无法读取导出的镜像文件")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 2<<30))
	if err != nil {
		return nil, "", errors.New("读取镜像文件失败")
	}
	_ = sc.Remove(remote)
	filename := "docker-images-" + time.Now().Format("20060102-150405") + ".tar"
	return data, filename, nil
}

// ImageImport 从 tar 包导入镜像。
func (s *DockerOperationService) ImageImport(ctx context.Context, assetType string, assetID, connectionID uint, content []byte) (string, error) {
	if len(content) == 0 || len(content) > 2<<30 {
		return "", errors.New("导入文件为空或过大")
	}
	remote := "/tmp/autoops-docker-" + dockerRandomSuffix() + ".tar"
	sc, client, err := s.sftpClient(ctx, assetType, assetID, connectionID)
	if err != nil {
		return "", err
	}
	defer sc.Close()
	defer client.Close()

	f, err := sc.Create(remote)
	if err != nil {
		return "", errors.New("无法在目标主机创建临时文件")
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		return "", errors.New("上传镜像文件失败")
	}
	f.Close()
	defer sc.Remove(remote)

	out, err := s.runLong(ctx, assetType, assetID, connectionID, "docker load -i "+remote, 8<<20)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
