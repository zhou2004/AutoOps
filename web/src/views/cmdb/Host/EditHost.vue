<template>
  <el-dialog
    :title='$tr("编辑主机")'
    v-model="dialogVisible"
    width="min(760px, 94vw)"
    @close="handleClose"
  >
    <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
      <el-row>
        <el-col :span="12">
          <el-form-item :label='$tr("主机名称")' prop="hostName">
            <el-input v-model="form.hostName" :placeholder='$tr("请输入主机名称")' />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item :label='$tr("所属分组")' prop="groupId">
            <el-select
              v-model="form.groupId"
              :placeholder='$tr("请选择分组")'
              style="width: 100%"
              clearable
              filterable
              @change="handleGroupChange"
            >
              <el-option
                v-for="group in selectableGroups"
                :key="group.id"
                :label="$tr(group.displayName)"
                :value="group.id"
                :class="
                  group.isBusinessGroup
                    ? 'business-group-option'
                    : 'sub-group-option'
                "
              >
                <div class="group-option-content">
                  <span
                    :class="
                      group.isBusinessGroup
                        ? 'business-group-text'
                        : 'sub-group-text'
                    "
                  >
                    {{ group.displayName }}
                  </span>
                  <el-tag
                    v-if="group.isBusinessGroup"
                    type="success"
                    size="small"
                    class="business-tag"
                  >
                    {{ $tr("默认") }}
                  </el-tag>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
      <div class="asset-form-grid">
        <el-form-item :label='$tr("操作系统")' prop="osType"
          ><el-select
            v-model="form.osType"
            @change="canBindConnection && (form.skipSSH = form.osType === 'windows')"
            ><el-option label="Linux" value="linux" /><el-option
              label="Windows"
              value="windows" /></el-select
        ></el-form-item>
        <el-form-item :label='$tr("系统版本")'
          ><el-input
            v-model="form.os"
            :placeholder='$tr("如 Ubuntu 24.04 / Windows Server 2022")'
        /></el-form-item>
      </div>
      <el-form-item :label='$tr("业务地址")' prop="privateIp"
        ><el-input
          v-model="form.privateIp"
          :placeholder='$tr("IP 地址（独立于远程连接档案）")'
      /></el-form-item>
      <el-form-item :label='$tr("接入方式")'
        ><el-switch
          v-model="form.skipSSH" :disabled="!canBindConnection"
          :active-text='$tr("先登记资产，稍后配置连接")'
        />
        <p class="asset-field-help">
          {{ $tr("连接地址与凭据由管理员配置；操作人员获得资产授权后即可使用连接。") }}
        </p></el-form-item
      >
      <el-row>
        <el-col :span="24">
          <el-form-item
            v-if="!form.skipSSH"
            :label='$tr("SSH连接")'
            prop="sshConnection"
          >
            <div style="display: flex; align-items: center">
              <el-input
                v-model="form.username" :disabled="!canBindConnection"
                :placeholder='$tr("用户名")'
                style="width: 120px; margin-right: 8px"
              />
              <span style="margin: 0 4px">@</span>
              <el-input
                v-model="form.ip" :disabled="!canBindConnection"
                :placeholder='$tr("IP地址")'
                style="width: 150px; margin-right: 8px"
              />
              <span style="margin: 0 4px">-p</span>
              <el-input
                v-model="form.port" :disabled="!canBindConnection"
                :placeholder='$tr("端口")'
                style="width: 100px"
              />
            </div>
          </el-form-item>
        </el-col>
      </el-row>
      <el-row>
        <el-col :span="12">
          <el-form-item v-if="!form.skipSSH" :label='$tr("认证凭据")' prop="authId">
            <el-select
              v-model="form.authId" :disabled="!canBindConnection"
              :placeholder='$tr("请选择认证凭据")'
              style="width: 100%"
              filterable
              clearable
            >
              <el-option
                v-for="item in authList"
                :key="item.id"
                :label="$tr(item.name)"
                :value="item.id"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item :label='$tr("主机类型")' prop="vendor">
            <el-select
              v-model="form.vendor"
              :placeholder='$tr("请选择主机类型")'
              style="width: 100%"
            >
              <el-option :value="1" :label="$tr('自建主机')">
                <div style="display: flex; align-items: center; gap: 8px">
                  <el-icon :size="18" color="#409EFF"
                    ><OfficeBuilding
                  /></el-icon>
                  <span>{{ $tr("自建主机") }}</span>
                </div>
              </el-option>
              <el-option :value="2" :label="$tr('阿里云')">
                <div style="display: flex; align-items: center; gap: 8px">
                  <img
                    :src="$getAssetUrl('@/assets/image/aliyun.png')"
                    style="width: 18px; height: 18px"
                  />
                  <span>{{ $tr("阿里云") }}</span>
                </div>
              </el-option>
              <el-option :value="3" :label="$tr('腾讯云')">
                <div style="display: flex; align-items: center; gap: 8px">
                  <img
                    :src="$getAssetUrl('@/assets/image/tengxun.png')"
                    style="width: 18px; height: 18px"
                  />
                  <span>{{ $tr("腾讯云") }}</span>
                </div>
              </el-option>
              <el-option :value="4" :label="$tr('百度云')">
                <div style="display: flex; align-items: center; gap: 8px">
                  <img
                    :src="$getAssetUrl('@/assets/image/baidu.svg')"
                    style="width: 18px; height: 18px"
                  />
                  <span>{{ $tr("百度云") }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
      <el-row>
        <el-col :span="24">
          <el-form-item :label='$tr("备注")' prop="remark">
            <el-input
              v-model="form.remark"
              type="textarea"
              :placeholder='$tr("请输入备注信息")'
            />
          </el-form-item>
        </el-col>
      </el-row>
      <span slot="footer" class="dialog-footer">
        <el-button type="primary" @click="submitForm">{{ $tr("确 定") }}</el-button>
        <el-button @click="dialogVisible = false">{{ $tr("取 消") }}</el-button>
      </span>
    </el-form>
  </el-dialog>
</template>

<script>
import { tr } from "@/locales";
import { platformAccess } from "@/utils/platformAccess";

export default {
  name: "EditHost",
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    hostInfo: {
      type: Object,
      required: true
    },
    groupList: {
      type: Array,
      required: true
    },
    authList: {
      type: Array,
      required: true
    }
  },
  data() {
    return {
      dialogVisible: this.visible,
      formReady: false,
      form: {
        id: "",
        hostName: "",
        osType: "linux",
        os: "",
        privateIp: "",
        skipSSH: false,
        groupId: "",
        remark: "",
        ip: "",
        port: 22,
        username: "",
        authId: "",
        vendor: 1
      },
      rules: {
        hostName: [
          { required: true, get message() { return tr("请输入主机名称"); }, trigger: "blur" }
        ],
        ip: [
          { required: true, get message() { return tr("请输入IP地址"); }, trigger: "blur" },
          {
            pattern:
              /^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$/,
            get message() { return tr("请输入有效的IP地址"); },
            trigger: "blur"
          }
        ],
        port: [
          { required: true, get message() { return tr("请输入端口号"); }, trigger: "blur" },
          { type: "number", get message() { return tr("端口号必须为数字"); }, trigger: "blur" },
          { min: 1, max: 65535, get message() { return tr("端口号范围1-65535"); }, trigger: "blur" }
        ],
        username: [
          { required: true, get message() { return tr("请输入连接用户名"); }, trigger: "blur" },
          { min: 1, max: 32, get message() { return tr("用户名长度1-32个字符"); }, trigger: "blur" }
        ],
        authId: [
          {
            validator: (rule, value, callback) =>
              callback(
                !this.form.skipSSH && !value
                  ? new Error(tr("请选择认证凭据"))
                  : undefined
              ),
            trigger: "change"
          }
        ],
        groupId: [
          { required: true, get message() { return tr("请选择所属分组"); }, trigger: "change" }
        ],
        vendor: [
          { required: true, get message() { return tr("请选择主机类型"); }, trigger: "change" }
        ]
      }
    };
  },
  computed: {
    canBindConnection() { return platformAccess.isAdmin; },
    // 生成可选择的分组列表：业务组 + 所有二级分组
    selectableGroups() {
      const groups = [];

      // 遍历分组树，提取业务组和二级分组
      const extractGroups = (groupList, parentName = "") => {
        groupList.forEach(group => {
          // 如果是业务组，直接添加
          if (group.name === "业务组" || group.isDefault) {
            groups.push({
              id: group.id,
              name: group.name,
              displayName: group.name,
              isBusinessGroup: true
            });
          }

          // 如果有子分组，遍历子分组（二级）
          if (group.children && group.children.length > 0) {
            group.children.forEach(subGroup => {
              // 只有叶子节点（没有子节点）才可选
              if (!subGroup.children || subGroup.children.length === 0) {
                groups.push({
                  id: subGroup.id,
                  name: subGroup.name,
                  displayName: subGroup.name,
                  parentName: group.name,
                  isBusinessGroup: false
                });
              }
            });
          }
        });
      };

      extractGroups(this.groupList);
      return groups;
    }
  },
  watch: {
    visible(newVal) {
      this.dialogVisible = newVal;
      if (newVal) {
        this.$nextTick(() => {
          this.initForm();
        });
      }
    },
    hostInfo: {
      deep: true,
      immediate: true,
      handler(newVal) {
        if (newVal) {
          this.form = {
            id: Number(newVal.id) || 0,
            hostName: newVal.hostName || "",
            osType:
              newVal.osType ||
              (String(newVal.os || "")
                .toLowerCase()
                .includes("windows")
                ? "windows"
                : "linux"),
            os: newVal.os || "",
            privateIp: newVal.privateIp || "",
            skipSSH: !newVal.sshIp,
            groupId:
              Number(newVal.groupId) ||
              (this.groupList.length > 0 ? Number(this.groupList[0].id) : ""),
            remark: newVal.remark || "",
            ip: newVal.sshIp || newVal.ip || "",
            port: newVal.sshPort || newVal.port || 22,
            username: newVal.sshName || newVal.username || "root",
            authId:
              Number(newVal.sshKeyId || newVal.authId) ||
              (this.authList.length > 0 ? Number(this.authList[0].id) : ""),
            vendor: Number(newVal.vendor) || 1
          };
        }
      }
    }
  },
  mounted() {
    if (this.visible) {
      this.initForm();
    }
  },
  methods: {
    async initForm() {
      await this.$nextTick();
      this.formReady = !!this.$refs.formRef;
      if (!this.formReady) {
        console.error("表单初始化失败", {
          refs: this.$refs,
          formRef: this.$refs.formRef
        });
      }
    },
    handleClose() {
      if (this.formReady) {
        this.$refs.formRef.resetFields();
      }
      this.$emit("close");
    },
    handleGroupChange(value) {
      if (value) {
        this.form.groupId = value;
      } else {
        // 清空时设置默认分组
        const defaultGroup = this.selectableGroups.find(
          group => group.isBusinessGroup
        );
        if (defaultGroup) {
          this.form.groupId = defaultGroup.id;
        }
      }
    },
    async submitForm() {
      try {
        // 确保表单组件已挂载
        await this.$nextTick();

        // 直接检查表单引用
        if (!this.$refs.formRef || !this.$refs.formRef.validate) {
          console.error("表单引用无效:", this.$refs.formRef);
          return this.$message.error(tr("表单组件未准备好，请稍后再试"));
        }

        // 直接调用validate方法
        return new Promise(resolve => {
          this.$refs.formRef.validate(valid => {
            if (valid) {
              // 验证凭据是否存在
              if (
                !this.form.skipSSH &&
                (!this.form.ip?.trim() || !this.form.username?.trim())
              ) {
                this.$message.error(tr("请填写 SSH 地址和用户名"));
                return;
              }
              const authExists = this.authList.some(
                auth => auth.id === this.form.authId
              );
              if (!this.form.skipSSH && this.canBindConnection && !authExists) {
                this.$message.error(
                  tr("选择的认证凭据不存在，请刷新凭据列表后重试")
                );
                return resolve(false);
              }

              // 确保端口为数字
              const port = Number(this.form.port);
              if (
                !this.form.skipSSH &&
                (isNaN(port) || port < 1 || port > 65535)
              ) {
                this.$message.error(tr("端口号必须为1-65535之间的数字"));
                return resolve(false);
              }

              // 转换数据格式为后端期望的结构
              const requestData = {
                id: Number(this.form.id),
                groupId: Number(this.form.groupId),
                hostName: this.form.hostName.trim(),
                remark: this.form.remark,
                osType: this.form.osType,
                os: this.form.os?.trim() || "",
                privateIp: this.form.privateIp?.trim() || "",
                skipSSH: this.form.skipSSH,
                sshIp: this.form.skipSSH ? "" : this.form.ip.trim(),
                sshKeyId: this.form.skipSSH ? 0 : Number(this.form.authId),
                sshName: this.form.skipSSH ? "" : this.form.username.trim(),
                sshPort: this.form.skipSSH ? 0 : port,
                vendor: Number(this.form.vendor)
              };

              this.$emit("submit", requestData);
              this.dialogVisible = false;
              resolve(true);
            } else {
              this.$message.error(tr("请填写完整的表单信息"));
              resolve(false);
            }
          });
        });
      } catch (error) {
        console.error("表单提交失败:", error);
        this.$message.error(`表单提交失败: ${error.message}`);
        return false;
      }
    }
  }
};
</script>

<style scoped>
/* 分组选择器样式 */
.group-option-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.business-group-option {
  background-color: #f0f9ff !important;
}

.business-group-text {
  color: #059669 !important;
  font-weight: 600 !important;
}

.sub-group-text {
  color: #374151 !important;
}

.business-tag {
  margin-left: 8px !important;
}

.sub-group-option:hover {
  background-color: #f3f4f6 !important;
}

.business-group-option:hover {
  background-color: #e0f2fe !important;
}
</style>
