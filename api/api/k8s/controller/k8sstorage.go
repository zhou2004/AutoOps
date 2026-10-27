package controller

import (
	"strconv"

	"dodevops-api/api/k8s/model"
	"dodevops-api/api/k8s/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// K8sStorageController K8s存储管理控制器
type K8sStorageController struct {
	service service.IK8sStorageService
}

func NewK8sStorageController(db *gorm.DB) *K8sStorageController {
	return &K8sStorageController{
		service: service.NewK8sStorageService(db),
	}
}

// ===================== PVC 管理 =====================

// GetPVCs 获取PVC列表
// @Summary 获取PVC列表
// @Description 获取指定命名空间的PVC列表
// @Tags K8s存储管理-PVC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param namespaceName path string true "命名空间名称"
// @Success 200 {object} result.Result{data=model.PVCListResponse}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/namespaces/{namespaceName}/pvcs [get]
func (ctrl *K8sStorageController) GetPVCs(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	ctrl.service.GetPVCs(c, uint(clusterId), namespaceName)
}

// GetPVCDetail 获取PVC详情
// @Summary 获取PVC详情
// @Description 获取指定PVC的详细信息
// @Tags K8s存储管理-PVC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param namespaceName path string true "命名空间名称"
// @Param pvcName path string true "PVC名称"
// @Success 200 {object} result.Result{data=model.PVCDetail}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/namespaces/{namespaceName}/pvcs/{pvcName} [get]
func (ctrl *K8sStorageController) GetPVCDetail(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	pvcName := c.Param("pvcName")
	if pvcName == "" {
		result.Failed(c, 400, "PVC名称不能为空")
		return
	}

	ctrl.service.GetPVCDetail(c, uint(clusterId), namespaceName, pvcName)
}

// CreatePVC 创建PVC
// @Summary 创建PVC
// @Description 在指定命名空间中创建新的PVC
// @Tags K8s存储管理-PVC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param namespaceName path string true "命名空间名称"
// @Param pvc body model.CreatePVCRequest true "PVC配置"
// @Success 200 {object} result.Result{data=model.K8sPersistentVolumeClaim}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/namespaces/{namespaceName}/pvcs [post]
func (ctrl *K8sStorageController) CreatePVC(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	var req model.CreatePVCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.CreatePVC(c, uint(clusterId), namespaceName, &req)
}

// UpdatePVC 更新PVC
// @Summary 更新PVC
// @Description 更新指定的PVC配置
// @Tags K8s存储管理-PVC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param namespaceName path string true "命名空间名称"
// @Param pvcName path string true "PVC名称"
// @Param pvc body model.UpdatePVCRequest true "更新配置"
// @Success 200 {object} result.Result{data=model.K8sPersistentVolumeClaim}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/namespaces/{namespaceName}/pvcs/{pvcName} [put]
func (ctrl *K8sStorageController) UpdatePVC(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	pvcName := c.Param("pvcName")
	if pvcName == "" {
		result.Failed(c, 400, "PVC名称不能为空")
		return
	}

	var req model.UpdatePVCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.UpdatePVC(c, uint(clusterId), namespaceName, pvcName, &req)
}

// DeletePVC 删除PVC
// @Summary 删除PVC
// @Description 删除指定的PVC
// @Tags K8s存储管理-PVC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param namespaceName path string true "命名空间名称"
// @Param pvcName path string true "PVC名称"
// @Success 200 {object} result.Result{data=string}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/namespaces/{namespaceName}/pvcs/{pvcName} [delete]
func (ctrl *K8sStorageController) DeletePVC(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	pvcName := c.Param("pvcName")
	if pvcName == "" {
		result.Failed(c, 400, "PVC名称不能为空")
		return
	}

	ctrl.service.DeletePVC(c, uint(clusterId), namespaceName, pvcName)
}

// GetPVCYaml 获取PVC的YAML配置
// @Summary 获取PVC的YAML配置
// @Description 获取指定PVC的完整YAML配置
// @Tags K8s存储管理-PVC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param namespaceName path string true "命名空间名称"
// @Param pvcName path string true "PVC名称"
// @Success 200 {object} result.Result{data=map[string]interface{}}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/namespaces/{namespaceName}/pvcs/{pvcName}/yaml [get]
func (ctrl *K8sStorageController) GetPVCYaml(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	pvcName := c.Param("pvcName")
	if pvcName == "" {
		result.Failed(c, 400, "PVC名称不能为空")
		return
	}

	ctrl.service.GetPVCYaml(c, uint(clusterId), namespaceName, pvcName)
}

// UpdatePVCYaml 通过YAML更新PVC
// @Summary 通过YAML更新PVC
// @Description 通过提供的YAML内容更新PVC配置
// @Tags K8s存储管理-PVC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param namespaceName path string true "命名空间名称"
// @Param pvcName path string true "PVC名称"
// @Param yaml body map[string]interface{} true "YAML内容"
// @Success 200 {object} result.Result{data=string}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/namespaces/{namespaceName}/pvcs/{pvcName}/yaml [put]
func (ctrl *K8sStorageController) UpdatePVCYaml(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	pvcName := c.Param("pvcName")
	if pvcName == "" {
		result.Failed(c, 400, "PVC名称不能为空")
		return
	}

	var yamlData map[string]interface{}
	if err := c.ShouldBindJSON(&yamlData); err != nil {
		result.Failed(c, 400, "YAML内容格式错误: "+err.Error())
		return
	}

	ctrl.service.UpdatePVCYaml(c, uint(clusterId), namespaceName, pvcName, yamlData)
}

// ===================== PV 管理 =====================

// GetPVs 获取PV列表
// @Summary 获取PV列表
// @Description 获取集群中的PV列表
// @Tags K8s存储管理-PV
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result{data=model.PVListResponse}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/pvs [get]
func (ctrl *K8sStorageController) GetPVs(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	ctrl.service.GetPVs(c, uint(clusterId))
}

// GetPVDetail 获取PV详情
// @Summary 获取PV详情
// @Description 获取指定PV的详细信息
// @Tags K8s存储管理-PV
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param pvName path string true "PV名称"
// @Success 200 {object} result.Result{data=model.PVDetail}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/pvs/{pvName} [get]
func (ctrl *K8sStorageController) GetPVDetail(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	pvName := c.Param("pvName")
	if pvName == "" {
		result.Failed(c, 400, "PV名称不能为空")
		return
	}

	ctrl.service.GetPVDetail(c, uint(clusterId), pvName)
}

// CreatePV 创建PV
// @Summary 创建PV
// @Description 在集群中创建新的PV
// @Tags K8s存储管理-PV
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param pv body model.CreatePVRequest true "PV配置"
// @Success 200 {object} result.Result{data=model.K8sPersistentVolume}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/pvs [post]
func (ctrl *K8sStorageController) CreatePV(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	var req model.CreatePVRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.CreatePV(c, uint(clusterId), &req)
}

// UpdatePV 更新PV
// @Summary 更新PV
// @Description 更新指定的PV配置
// @Tags K8s存储管理-PV
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param pvName path string true "PV名称"
// @Param pv body model.UpdatePVRequest true "更新配置"
// @Success 200 {object} result.Result{data=model.K8sPersistentVolume}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/pvs/{pvName} [put]
func (ctrl *K8sStorageController) UpdatePV(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	pvName := c.Param("pvName")
	if pvName == "" {
		result.Failed(c, 400, "PV名称不能为空")
		return
	}

	var req model.UpdatePVRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.UpdatePV(c, uint(clusterId), pvName, &req)
}

// DeletePV 删除PV
// @Summary 删除PV
// @Description 删除指定的PV
// @Tags K8s存储管理-PV
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param pvName path string true "PV名称"
// @Success 200 {object} result.Result{data=string}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/pvs/{pvName} [delete]
func (ctrl *K8sStorageController) DeletePV(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	pvName := c.Param("pvName")
	if pvName == "" {
		result.Failed(c, 400, "PV名称不能为空")
		return
	}

	ctrl.service.DeletePV(c, uint(clusterId), pvName)
}

// GetPVYaml 获取PV的YAML配置
// @Summary 获取PV的YAML配置
// @Description 获取指定PV的完整YAML配置
// @Tags K8s存储管理-PV
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param pvName path string true "PV名称"
// @Success 200 {object} result.Result{data=map[string]interface{}}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/pvs/{pvName}/yaml [get]
func (ctrl *K8sStorageController) GetPVYaml(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	pvName := c.Param("pvName")
	if pvName == "" {
		result.Failed(c, 400, "PV名称不能为空")
		return
	}

	ctrl.service.GetPVYaml(c, uint(clusterId), pvName)
}

// UpdatePVYaml 通过YAML更新PV
// @Summary 通过YAML更新PV
// @Description 通过提供的YAML内容更新PV配置
// @Tags K8s存储管理-PV
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param pvName path string true "PV名称"
// @Param yaml body map[string]interface{} true "YAML内容"
// @Success 200 {object} result.Result{data=string}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/pvs/{pvName}/yaml [put]
func (ctrl *K8sStorageController) UpdatePVYaml(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	pvName := c.Param("pvName")
	if pvName == "" {
		result.Failed(c, 400, "PV名称不能为空")
		return
	}

	var yamlData map[string]interface{}
	if err := c.ShouldBindJSON(&yamlData); err != nil {
		result.Failed(c, 400, "YAML内容格式错误: "+err.Error())
		return
	}

	ctrl.service.UpdatePVYaml(c, uint(clusterId), pvName, yamlData)
}

// ===================== StorageClass 管理 =====================

// GetStorageClasses 获取存储类列表
// @Summary 获取存储类列表
// @Description 获取集群中的存储类列表
// @Tags K8s存储管理-StorageClass
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result{data=model.StorageClassListResponse}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/storageclasses [get]
func (ctrl *K8sStorageController) GetStorageClasses(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	ctrl.service.GetStorageClasses(c, uint(clusterId))
}

// GetStorageClassDetail 获取存储类详情
// @Summary 获取存储类详情
// @Description 获取指定存储类的详细信息
// @Tags K8s存储管理-StorageClass
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param storageClassName path string true "存储类名称"
// @Success 200 {object} result.Result{data=model.StorageClassDetail}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/storageclasses/{storageClassName} [get]
func (ctrl *K8sStorageController) GetStorageClassDetail(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	storageClassName := c.Param("storageClassName")
	if storageClassName == "" {
		result.Failed(c, 400, "存储类名称不能为空")
		return
	}

	ctrl.service.GetStorageClassDetail(c, uint(clusterId), storageClassName)
}

// CreateStorageClass 创建存储类
// @Summary 创建存储类
// @Description 在集群中创建新的存储类
// @Tags K8s存储管理-StorageClass
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param storageClass body model.CreateStorageClassRequest true "存储类配置"
// @Success 200 {object} result.Result{data=model.K8sStorageClass}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/storageclasses [post]
func (ctrl *K8sStorageController) CreateStorageClass(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	var req model.CreateStorageClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.CreateStorageClass(c, uint(clusterId), &req)
}

// UpdateStorageClass 更新存储类
// @Summary 更新存储类
// @Description 更新指定的存储类配置
// @Tags K8s存储管理-StorageClass
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param storageClassName path string true "存储类名称"
// @Param storageClass body model.UpdateStorageClassRequest true "更新配置"
// @Success 200 {object} result.Result{data=model.K8sStorageClass}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/storageclasses/{storageClassName} [put]
func (ctrl *K8sStorageController) UpdateStorageClass(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	storageClassName := c.Param("storageClassName")
	if storageClassName == "" {
		result.Failed(c, 400, "存储类名称不能为空")
		return
	}

	var req model.UpdateStorageClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.UpdateStorageClass(c, uint(clusterId), storageClassName, &req)
}

// DeleteStorageClass 删除存储类
// @Summary 删除存储类
// @Description 删除指定的存储类
// @Tags K8s存储管理-StorageClass
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param storageClassName path string true "存储类名称"
// @Success 200 {object} result.Result{data=string}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/storageclasses/{storageClassName} [delete]
func (ctrl *K8sStorageController) DeleteStorageClass(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	storageClassName := c.Param("storageClassName")
	if storageClassName == "" {
		result.Failed(c, 400, "存储类名称不能为空")
		return
	}

	ctrl.service.DeleteStorageClass(c, uint(clusterId), storageClassName)
}

// GetStorageClassYaml 获取存储类的YAML配置
// @Summary 获取存储类的YAML配置
// @Description 获取指定存储类的完整YAML配置
// @Tags K8s存储管理-StorageClass
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param storageClassName path string true "存储类名称"
// @Success 200 {object} result.Result{data=map[string]interface{}}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/storageclasses/{storageClassName}/yaml [get]
func (ctrl *K8sStorageController) GetStorageClassYaml(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	storageClassName := c.Param("storageClassName")
	if storageClassName == "" {
		result.Failed(c, 400, "存储类名称不能为空")
		return
	}

	ctrl.service.GetStorageClassYaml(c, uint(clusterId), storageClassName)
}

// UpdateStorageClassYaml 通过YAML更新存储类
// @Summary 通过YAML更新存储类
// @Description 通过提供的YAML内容更新存储类配置
// @Tags K8s存储管理-StorageClass
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户Token"
// @Param id path int true "集群ID"
// @Param storageClassName path string true "存储类名称"
// @Param yaml body map[string]interface{} true "YAML内容"
// @Success 200 {object} result.Result{data=string}
// @Failure 400 {object} result.Result
// @Failure 401 {object} result.Result
// @Failure 404 {object} result.Result
// @Failure 500 {object} result.Result
// @Router /k8s/cluster/{id}/storageclasses/{storageClassName}/yaml [put]
func (ctrl *K8sStorageController) UpdateStorageClassYaml(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	storageClassName := c.Param("storageClassName")
	if storageClassName == "" {
		result.Failed(c, 400, "存储类名称不能为空")
		return
	}

	var yamlData map[string]interface{}
	if err := c.ShouldBindJSON(&yamlData); err != nil {
		result.Failed(c, 400, "YAML内容格式错误: "+err.Error())
		return
	}

	ctrl.service.UpdateStorageClassYaml(c, uint(clusterId), storageClassName, yamlData)
}

// ===================== PVC 高阶功能 =====================

// ExpandPVC PVC扩容
func (ctrl *K8sStorageController) ExpandPVC(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	pvcName := c.Param("pvcName")
	if pvcName == "" {
		result.Failed(c, 400, "PVC名称不能为空")
		return
	}

	var req model.ExpandPVCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.ExpandPVC(c, uint(clusterId), namespaceName, pvcName, &req)
}

// ClonePVC PVC克隆（数据备份迁移）
func (ctrl *K8sStorageController) ClonePVC(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	pvcName := c.Param("pvcName")
	if pvcName == "" {
		result.Failed(c, 400, "PVC名称不能为空")
		return
	}

	var req model.PVCCloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.ClonePVC(c, uint(clusterId), namespaceName, pvcName, &req)
}

// ===================== StorageClass 高阶功能 =====================

// SetDefaultStorageClass 设置默认存储类
func (ctrl *K8sStorageController) SetDefaultStorageClass(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	storageClassName := c.Param("storageClassName")
	if storageClassName == "" {
		result.Failed(c, 400, "存储类名称不能为空")
		return
	}

	var req model.SetDefaultSCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.SetDefaultStorageClass(c, uint(clusterId), storageClassName, &req)
}

// GetStorageStats 获取存储统计信息
func (ctrl *K8sStorageController) GetStorageStats(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	ctrl.service.GetStorageStats(c, uint(clusterId))
}

// ===================== Ceph 管理 =====================

// GetCephStatus 获取 Ceph 集群状态概览
// @Summary 获取 Ceph 集群状态概览
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result{data=model.CephClusterStatus}
// @Router /k8s/cluster/{id}/ceph/status [get]
func (ctrl *K8sStorageController) GetCephStatus(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}
	ctrl.service.GetCephStatus(c, uint(clusterId))
}

// GetCephPools 获取 Ceph pool 列表
// @Summary 获取 Ceph pool 列表
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result{data=[]model.CephPool}
// @Router /k8s/cluster/{id}/ceph/pools [get]
func (ctrl *K8sStorageController) GetCephPools(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}
	ctrl.service.GetCephPools(c, uint(clusterId))
}

// UpdateCephPoolSize 调整 Ceph pool 副本数
// @Summary 调整 Ceph pool 副本数
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param poolName path string true "Pool 名称"
// @Param body body model.UpdateCephPoolSizeRequest true "目标副本数"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/pools/{poolName}/size [put]
func (ctrl *K8sStorageController) UpdateCephPoolSize(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}
	poolName := c.Param("poolName")
	var req model.UpdateCephPoolSizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.UpdateCephPoolSize(c, uint(clusterId), poolName, &req)
}

// GetCephOSDs 获取 Ceph OSD 状态列表
// @Summary 获取 Ceph OSD 状态列表
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result{data=[]model.CephOSD}
// @Router /k8s/cluster/{id}/ceph/osds [get]
func (ctrl *K8sStorageController) GetCephOSDs(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}
	ctrl.service.GetCephOSDs(c, uint(clusterId))
}

// CreateCephPool 创建 Ceph pool
// @Summary 创建 Ceph pool
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param body body model.CreateCephPoolRequest true "Pool 信息"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/pools [post]
func (ctrl *K8sStorageController) CreateCephPool(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.CreateCephPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.CreateCephPool(c, clusterId, &req)
}

// DeleteCephPool 删除 Ceph pool
// @Summary 删除 Ceph pool
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param poolName path string true "Pool 名称"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/pools/{poolName} [delete]
func (ctrl *K8sStorageController) DeleteCephPool(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.DeleteCephPool(c, clusterId, c.Param("poolName"))
}

// UpdateCephPoolPGNum 调整 Ceph pool PG 数
// @Summary 调整 Ceph pool PG 数
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param poolName path string true "Pool 名称"
// @Param body body model.UpdateCephPoolPGNumRequest true "目标PG数"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/pools/{poolName}/pg-num [put]
func (ctrl *K8sStorageController) UpdateCephPoolPGNum(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.UpdateCephPoolPGNumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.UpdateCephPoolPGNum(c, clusterId, c.Param("poolName"), &req)
}

// UpdateCephPoolQuota 设置 Ceph pool 配额
// @Summary 设置 Ceph pool 配额
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param poolName path string true "Pool 名称"
// @Param body body model.UpdateCephPoolQuotaRequest true "配额信息"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/pools/{poolName}/quota [put]
func (ctrl *K8sStorageController) UpdateCephPoolQuota(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.UpdateCephPoolQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.UpdateCephPoolQuota(c, clusterId, c.Param("poolName"), &req)
}

// UpdateCephPoolCompression 设置 Ceph pool 压缩
// @Summary 设置 Ceph pool 压缩
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param poolName path string true "Pool 名称"
// @Param body body model.UpdateCephPoolCompressionRequest true "压缩配置"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/pools/{poolName}/compression [put]
func (ctrl *K8sStorageController) UpdateCephPoolCompression(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.UpdateCephPoolCompressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.UpdateCephPoolCompression(c, clusterId, c.Param("poolName"), &req)
}

// GetCephRBDImages 获取 Ceph RBD 镜像列表
// @Summary 获取 Ceph RBD 镜像列表
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result{data=[]model.CephRBDImage}
// @Router /k8s/cluster/{id}/ceph/rbd/images [get]
func (ctrl *K8sStorageController) GetCephRBDImages(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.GetCephRBDImages(c, clusterId)
}

// CreateCephRBDImage 创建 Ceph RBD 镜像
// @Summary 创建 Ceph RBD 镜像
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param body body model.CreateCephRBDImageRequest true "镜像信息"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/rbd/images [post]
func (ctrl *K8sStorageController) CreateCephRBDImage(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.CreateCephRBDImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.CreateCephRBDImage(c, clusterId, &req)
}

// GetCephRBDImageDetail 获取 Ceph RBD 镜像详情
// @Summary 获取 Ceph RBD 镜像详情
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Success 200 {object} result.Result{data=model.CephRBDImage}
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image} [get]
func (ctrl *K8sStorageController) GetCephRBDImageDetail(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.GetCephRBDImageDetail(c, clusterId, c.Param("pool"), c.Param("image"))
}

// ResizeCephRBDImage 扩容/缩容 RBD 镜像
// @Summary 扩容/缩容 RBD 镜像
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Param body body model.ResizeCephRBDImageRequest true "目标容量"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image}/resize [put]
func (ctrl *K8sStorageController) ResizeCephRBDImage(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.ResizeCephRBDImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.ResizeCephRBDImage(c, clusterId, c.Param("pool"), c.Param("image"), &req)
}

// DeleteCephRBDImage 删除 RBD 镜像
// @Summary 删除 RBD 镜像
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image} [delete]
func (ctrl *K8sStorageController) DeleteCephRBDImage(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.DeleteCephRBDImage(c, clusterId, c.Param("pool"), c.Param("image"))
}

// ListCephRBDSnapshots 获取 RBD 镜像快照列表
// @Summary 获取 RBD 镜像快照列表
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Success 200 {object} result.Result{data=[]model.CephRBDSnapshot}
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image}/snapshots [get]
func (ctrl *K8sStorageController) ListCephRBDSnapshots(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.ListCephRBDSnapshots(c, clusterId, c.Param("pool"), c.Param("image"))
}

// CreateCephRBDSnapshot 创建 RBD 快照
// @Summary 创建 RBD 快照
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Param body body model.CreateCephRBDSnapshotRequest true "快照名"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image}/snapshots [post]
func (ctrl *K8sStorageController) CreateCephRBDSnapshot(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.CreateCephRBDSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.CreateCephRBDSnapshot(c, clusterId, c.Param("pool"), c.Param("image"), &req)
}

// DeleteCephRBDSnapshot 删除 RBD 快照
// @Summary 删除 RBD 快照
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Param snap path string true "快照名称"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image}/snapshots/{snap} [delete]
func (ctrl *K8sStorageController) DeleteCephRBDSnapshot(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.DeleteCephRBDSnapshot(c, clusterId, c.Param("pool"), c.Param("image"), c.Param("snap"))
}

// RollbackCephRBDImage 回滚 RBD 镜像到快照
// @Summary 回滚 RBD 镜像到快照
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Param snap path string true "快照名称"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image}/snapshots/{snap}/rollback [post]
func (ctrl *K8sStorageController) RollbackCephRBDImage(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.RollbackCephRBDImage(c, clusterId, c.Param("pool"), c.Param("image"), c.Param("snap"))
}

// CloneCephRBDImage 克隆 RBD 镜像
// @Summary 克隆 RBD 镜像（基于受保护快照）
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pool path string true "Pool 名称"
// @Param image path string true "镜像名称"
// @Param body body model.CloneCephRBDImageRequest true "克隆目标"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/rbd/images/{pool}/{image}/clone [post]
func (ctrl *K8sStorageController) CloneCephRBDImage(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	var req model.CloneCephRBDImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.CloneCephRBDImage(c, clusterId, c.Param("pool"), c.Param("image"), &req)
}

// GetCephFS 获取 CephFS 文件系统列表
// @Summary 获取 CephFS 文件系统列表
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result{data=[]model.CephFS}
// @Router /k8s/cluster/{id}/ceph/fs [get]
func (ctrl *K8sStorageController) GetCephFS(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.GetCephFS(c, clusterId)
}

// GetCephHealth 获取 Ceph 健康详情
// @Summary 获取 Ceph 健康详情
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/health [get]
func (ctrl *K8sStorageController) GetCephHealth(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.GetCephHealth(c, clusterId)
}

// GetCephVersions 获取 Ceph 组件版本分布
// @Summary 获取 Ceph 组件版本分布
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/versions [get]
func (ctrl *K8sStorageController) GetCephVersions(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.GetCephVersions(c, clusterId)
}

// GetCephPGStat 获取 Ceph PG 状态统计
// @Summary 获取 Ceph PG 状态统计
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/pg-stat [get]
func (ctrl *K8sStorageController) GetCephPGStat(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	ctrl.service.GetCephPGStat(c, clusterId)
}

// SetCephOSDStatus 标记 OSD in/out/down
// @Summary 标记 OSD in/out/down
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param osdId path int true "OSD ID"
// @Param action path string true "动作 in/out/down"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/osds/{osdId}/{action} [post]
func (ctrl *K8sStorageController) SetCephOSDStatus(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	osdId, err := strconv.Atoi(c.Param("osdId"))
	if err != nil {
		result.Failed(c, 400, "无效的OSD ID")
		return
	}
	ctrl.service.SetCephOSDStatus(c, clusterId, osdId, c.Param("action"))
}

// ReweightCephOSD 调整 OSD 权重
// @Summary 调整 OSD 权重
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param osdId path int true "OSD ID"
// @Param body body model.ReweightCephOSDRequest true "目标权重"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/osds/{osdId}/reweight [post]
func (ctrl *K8sStorageController) ReweightCephOSD(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	osdId, err := strconv.Atoi(c.Param("osdId"))
	if err != nil {
		result.Failed(c, 400, "无效的OSD ID")
		return
	}
	var req model.ReweightCephOSDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}
	ctrl.service.ReweightCephOSD(c, clusterId, osdId, &req)
}

// DeleteCephOSD 删除 OSD
// @Summary 删除 OSD（危险操作）
// @Tags K8s存储管理-Ceph
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param osdId path int true "OSD ID"
// @Success 200 {object} result.Result
// @Router /k8s/cluster/{id}/ceph/osds/{osdId} [delete]
func (ctrl *K8sStorageController) DeleteCephOSD(c *gin.Context) {
	clusterId, ok := ctrl.parseClusterId(c)
	if !ok {
		return
	}
	osdId, err := strconv.Atoi(c.Param("osdId"))
	if err != nil {
		result.Failed(c, 400, "无效的OSD ID")
		return
	}
	ctrl.service.DeleteCephOSD(c, clusterId, osdId)
}

// parseClusterId 解析集群ID（Ceph controller 通用）
func (ctrl *K8sStorageController) parseClusterId(c *gin.Context) (uint, bool) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return 0, false
	}
	return uint(clusterId), true
}

// MigratePVC 卷到卷数据迁移
func (ctrl *K8sStorageController) MigratePVC(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	var req model.MigratePVCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.MigratePVC(c, uint(clusterId), &req)
}

// SwitchWorkloadPVC 一键换卷
func (ctrl *K8sStorageController) SwitchWorkloadPVC(c *gin.Context) {
	clusterIdStr := c.Param("id")
	clusterId, err := strconv.Atoi(clusterIdStr)
	if err != nil || clusterId <= 0 {
		result.Failed(c, 400, "无效的集群ID")
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		result.Failed(c, 400, "命名空间名称不能为空")
		return
	}

	var req model.SwitchWorkloadPVCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "参数验证失败: "+err.Error())
		return
	}

	ctrl.service.SwitchWorkloadPVC(c, uint(clusterId), namespaceName, &req)
}
