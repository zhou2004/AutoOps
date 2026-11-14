// Package datacenterseed creates explicitly requested visualization test inventory.
// It is only called by the CLI; application startup never seeds sample assets.
package datacenterseed

import (
	"errors"
	"fmt"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"gorm.io/gorm"
)

const Marker = "autoops:idc-sample:v1"
const note = Marker + "；机房可视化测试数据，无真实连接，不代表生产设备。"

type Counts struct {
	Rooms    int `json:"rooms"`
	Cabinets int `json:"cabinets"`
	Servers  int `json:"servers"`
	Networks int `json:"networks"`
}
type Receipt struct {
	Created  Counts `json:"created"`
	Existing Counts `json:"existing"`
	RoomIDs  []uint `json:"roomIds"`
}

var locations = []struct {
	code, name, country, city string
	lon, lat                  float64
}{
	{"BJ", "北京 · 北方数据中心", "中国", "北京", 116.4074, 39.9042},
	{"SH", "上海 · 华东数据中心", "中国", "上海", 121.4737, 31.2304},
	{"SG", "新加坡 · 亚太数据中心", "新加坡", "新加坡", 103.8198, 1.3521},
	{"FRA", "法兰克福 · 欧洲数据中心", "德国", "法兰克福", 8.6821, 50.1109},
	{"IAD", "阿什本 · 北美数据中心", "美国", "阿什本", -77.4874, 39.0438},
}

// Seed is additive and transactional. A rerun preserves all edits to existing
// samples and fails on a colliding identifier owned by non-sample inventory.
func Seed(db *gorm.DB) (Receipt, error) {
	var out Receipt
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, loc := range locations {
			room := model.CmdbIDC{}
			name := "[测试] " + loc.name
			err := tx.Where("name = ?", name).First(&room).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				lon, lat := loc.lon, loc.lat
				room = model.CmdbIDC{Name: name, ShortName: "TEST-" + loc.code, Country: loc.country, City: loc.city, Longitude: &lon, Latitude: &lat, Address: "可视化测试园区", Level: "T3", Status: 1, Description: note}
				if err = dao.NewCmdbIDCDao(tx).Create(&room); err != nil {
					return err
				}
				out.Created.Rooms++
			} else if err != nil {
				return err
			} else {
				if room.Description != note {
					return fmt.Errorf("机房标识冲突: %s", name)
				}
				out.Existing.Rooms++
			}
			out.RoomIDs = append(out.RoomIDs, room.ID)
			for i := 0; i < 10; i++ {
				row := "A"
				if i >= 5 {
					row = "B"
				}
				rackName := fmt.Sprintf("TEST-%s-%s%02d", loc.code, row, i%5+1)
				rack := model.CmdbCabinet{}
				err = tx.Where("name = ?", rackName).First(&rack).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					x, z := float64(i%5)*1.1, float64(i/5)*2.7
					rack = model.CmdbCabinet{Name: rackName, IDCID: room.ID, Position: fmt.Sprintf("测试 %s 排 %d 号", row, i%5+1), LayoutX: &x, LayoutZ: &z, WidthMM: 600, DepthMM: 1000, UnitNum: 42, Status: 1, Remark: note}
					if err = dao.NewCmdbCabinetDao(tx).Create(&rack); err != nil {
						return err
					}
					out.Created.Cabinets++
				} else if err != nil {
					return err
				} else {
					if rack.Remark != note || rack.IDCID != room.ID {
						return fmt.Errorf("机柜标识冲突: %s", rackName)
					}
					out.Existing.Cabinets++
				}
				for n := 0; n < 13; n++ {
					sn := fmt.Sprintf("AUTOOPS-IDC-SAMPLE-%s-%02d", rackName, n+1)
					if n < 11 {
						asset := model.CmdbPhysicalMachine{}
						err = tx.Where("sn = ?", sn).First(&asset).Error
						if errors.Is(err, gorm.ErrRecordNotFound) {
							asset = model.CmdbPhysicalMachine{SN: sn, HostName: fmt.Sprintf("[测试] %s 计算节点 %02d", rackName, n+1), Brand: "SERVER", Model: "2U COMPUTE", IDCID: room.ID, CabinetID: rack.ID, UnitPosition: 1 + n*3, UnitHeight: 2, AssetStatus: 2, Status: 3, Remark: note}
							if err = dao.NewCmdbPhysicalMachineDao(tx).Create(&asset); err != nil {
								return err
							}
							out.Created.Servers++
						} else if err != nil {
							return err
						} else {
							if asset.Remark != note {
								return fmt.Errorf("设备标识冲突: %s", sn)
							}
							out.Existing.Servers++
						}
					} else {
						asset := model.CmdbNetworkDevice{}
						err = tx.Where("sn = ?", sn).First(&asset).Error
						if errors.Is(err, gorm.ErrRecordNotFound) {
							asset = model.CmdbNetworkDevice{SN: sn, Name: fmt.Sprintf("[测试] %s 汇聚交换机 %d", rackName, n-10), DeviceType: 2, Brand: "NETWORK", Model: "24 PORT GbE", PortNum: 24, IDCID: room.ID, CabinetID: rack.ID, UnitPosition: 39 + n - 11, UnitHeight: 1, AssetStatus: 2, Status: 3, Remark: note}
							if err = dao.NewCmdbNetworkDeviceDao(tx).Create(&asset); err != nil {
								return err
							}
							out.Created.Networks++
						} else if err != nil {
							return err
						} else {
							if asset.Remark != note {
								return fmt.Errorf("设备标识冲突: %s", sn)
							}
							out.Existing.Networks++
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return Receipt{}, err
	}
	return out, nil
}
