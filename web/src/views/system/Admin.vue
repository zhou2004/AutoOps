<template>
  <div class="modern-admin-container">
    <!-- 主内容区域 -->
    <div class="glass-card main-card">
      <template v-if="false"></template>
      <div class="card-header">
        <span class="main-title">用户管理系统</span>
      </div>
      <!-- 左右布局容器 -->
      <div class="user-management-container">
        <!-- 左侧部门树 -->
        <div class="dept-tree-section">
          <div class="glass-card tree-card">
            <div class="card-header">
              <h3 class="section-title">部门选择</h3>
            </div>
            <el-tree
                :data="deptList"
                :props="defaultProps"
                node-key="id"
                default-expand-all
                :highlight-current="true"
                @node-click="handleDeptClick"
                @node-expand="handleNodeExpand"
                @node-collapse="handleNodeCollapse"
                class="modern-tree"
            >
              <template v-slot="{ node, data }">
                <span :class="{ 'font-weight-bold': !data.parentId }" class="tree-node">
                  <!-- 一级部门：根据展开状态切换图标 -->
                  <template v-if="!data.parentId">
                    <el-icon v-if="expandedKeys.includes(node.key)" style="margin-right: 5px"><FolderOpened /></el-icon>
                    <el-icon v-else style="margin-right: 5px"><Folder /></el-icon>
                  </template>
                  <!-- 二级部门：固定使用DocumentRemove图标 -->
                  <template v-else>
                    <el-icon style="margin-right: 5px"><Folder /></el-icon>
                  </template>
                  {{ node.label }}
                </span>
              </template>
            </el-tree>
          </div>
        </div>
        
        <!-- 右侧用户管理区域 -->
        <div class="user-table-section">
          <!-- 搜索表单 -->
          <div class="glass-card search-card">
            <el-form :inline="true" :model="queryParams" class="modern-form">
              <el-form-item label="用户账号" prop="username">
                <el-input
                    size="small"
                    placeholder="请输入用户账号"
                    clearable
                    v-model="queryParams.username"
                    @keyup.enter="handleQuery"
                    class="modern-input"
                />
              </el-form-item>
              <el-form-item label="账号状态" prop="status"  style="width: 150px;" >
                <el-select size="small" placeholder="账号状态" v-model="queryParams.status" class="modern-select">
                  <el-option v-for="item in statusList" :key="item.value" :label="item.label" :value="item.value"></el-option>
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" size="small" @click="handleQuery" class="modern-btn primary-btn">
                  <el-icon><Search /></el-icon>
                  搜索
                </el-button>
                <el-button size="small" @click="resetQuery" class="modern-btn reset-btn">
                  <el-icon><Refresh /></el-icon>
                  重置
                </el-button>
                <el-button type="success" size="small" @click="addDialogVisible = true" v-authority="['base:admin:add']" class="modern-btn success-btn">
                  <el-icon><Plus /></el-icon>
                  新增
                </el-button>
              </el-form-item>
            </el-form>
          </div>

          <!-- 用户表格 -->
          <div class="glass-card table-card">
            <div class="card-header">
              <h3 class="section-title">用户列表</h3>
            </div>
            <el-table
                v-loading="Loading"
                :data="adminList"
                class="modern-table"
                :row-class-name="tableRowClassName"
            >
              <el-table-column label="ID" prop="id" v-if="false" />

              <el-table-column label="用户账号" prop="username"  />
              <el-table-column label="用户头像" prop="icon" >
                <template v-slot="scope">
                  <el-avatar shape="circle" :src="scope.row.icon" class="modern-avatar"></el-avatar>
                </template>
              </el-table-column>
              <el-table-column label="用户昵称" prop="nickname" />
              <el-table-column label="角色名称" prop="roleName" >
                <template v-slot="scope">
                  <el-tag type="success" class="modern-tag">{{ scope.row.roleName }}</el-tag>
                </template>
              </el-table-column>

              <el-table-column label="岗位" >
                <template v-slot="scope">
                  <div class="modern-text">{{ scope.row.postName }}</div>
                </template>
              </el-table-column>
              <el-table-column label="账号状态" >
                <template v-slot="scope">
                  <el-switch
                      v-model="scope.row.status"
                      :active-value="1"
                      :inactive-value="2"
                      active-color="#67C23A"
                      inactive-color="#F56C6C"
                      active-text="启用"
                      inactive-text="停用"
                      @change="adminUpdateStatus(scope.row)"
                      class="modern-switch"
                  ></el-switch>
                </template>
              </el-table-column>

              <el-table-column label="操作" fixed="right" width="200">
                <template v-slot="scope">
                  <div class="operation-buttons">
                    <el-tooltip content="编辑" placement="top">
                      <el-button
                        type="warning"
                        size="small"
                        circle
                        @click="showEditAdminDialog(scope.row.id)"
                        v-authority="['base:admin:edit']"
                      >
                        <el-icon><Edit /></el-icon>
                      </el-button>
                    </el-tooltip>
                    <el-tooltip content="删除" placement="top">
                      <el-button
                        type="danger"
                        size="small"
                        circle
                        @click="handleAdminDelete(scope.row)"
                        v-authority="['base:admin:delete']"
                      >
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </el-tooltip>
                    <el-tooltip content="重置密码" placement="top">
                      <el-button
                        type="info"
                        size="small"
                        circle
                        @click="handleResetPwd(scope.row)"
                        v-authority="['base:admin:reset']"
                      >
                        <el-icon><Key /></el-icon>
                      </el-button>
                    </el-tooltip>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <!-- 分页 -->
          <div class="glass-card pagination-card">
            <el-pagination
                @size-change="handleSizeChange"
                @current-change="handleCurrentChange"
                :current-page="queryParams.pageNum"
                :page-sizes="[10, 50, 100, 500, 1000]"
                :page-size="queryParams.pageSize"
                layout="total, sizes, prev, pager, next, jumper"
                :total="total"
                class="modern-pagination"
            ></el-pagination>
          </div>
        </div>
      </div>
    </div>

    <!-- 新增用户对话框 -->
    <el-dialog title="新增用户" v-model="addDialogVisible" width="40%" @close="addDialogClosed" class="modern-dialog">
      <el-form :model="addForm" :rules="addFormRules" ref="addFormRefForm" label-width="80px" class="modern-dialog-form">
        <el-row>
          <el-col :span="12">
            <el-form-item label="用户名称" prop="username">
              <el-input v-model="addForm.username" placeholder="请输入英文名称" maxlength="30" class="modern-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户密码" prop="password">
              <el-input v-model="addForm.password" placeholder="请输入用户密码" type="password" maxlength="20" show-password class="modern-input" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="用户昵称" prop="nickname">
              <el-input v-model="addForm.nickname" placeholder="请输入中文昵称" maxlength="30" class="modern-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item size="small" label="归属部门" prop="deptId">
              <treeselect v-model="addForm.deptId" :options="deptList" :show-count="true"
                          placeholder="请选择归属部门" class="modern-treeselect" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="手机号码" prop="phone">
              <el-input v-model="addForm.phone" placeholder="请输入手机号码" maxlength="11" class="modern-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户邮箱" prop="email">
              <el-input v-model="addForm.email" placeholder="请输入邮箱" maxlength="50" class="modern-input" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="用户状态" prop="status">
              <el-radio-group v-model="addForm.status" class="modern-radio-group">
                <el-radio :label="1" class="modern-radio">正常</el-radio>
                <el-radio :label="2" class="modern-radio">停用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="用户岗位" prop="postId">
              <el-select placeholder="请选择岗位" v-model="addForm.postId" class="modern-select">
                <el-option v-for="item in postList" :key="item.id" :label="item.postName" :value="item.id"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户角色" prop="roleId">
              <el-select placeholder="请选择角色" v-model="addForm.roleId" class="modern-select">
                <el-option v-for="item in roleList" :key="item.id" :label="item.roleName" :value="item.id"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="24">
            <el-form-item label="个人简介" prop="note">
              <el-input v-model="addForm.note" type="textarea" placeholder="请输入内容" class="modern-textarea"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
        <span slot="footer" class="dialog-footer">
          <el-button type="primary" @click="addAdmin" class="modern-btn primary-btn">确 定</el-button>
          <el-button type="info" @click="addDialogVisible = false" class="modern-btn secondary-btn">取 消</el-button>
        </span>
      </el-form>
    </el-dialog>

    <!-- 编辑用户对话框 -->
    <el-dialog title="修改用户" v-model="editDialogVisible" width="40%" @close="editDialogClosed" class="modern-dialog">
      <el-form :model="adminInfo" :rules="editFormRules" ref="editFormRefForm" label-width="80px" class="modern-dialog-form">
        <el-row>
          <el-col :span="12">
            <el-form-item label="用户昵称" prop="nickname">
              <el-input v-model="adminInfo.nickname" placeholder="请输入用户昵称" maxlength="30" class="modern-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item size="small" label="归属部门" prop="deptId">
              <treeselect v-model="adminInfo.deptId" :options="deptList" :show-count="true"
                          placeholder="请选择归属部门" class="modern-treeselect" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="手机号码" prop="phone">
              <el-input v-model="adminInfo.phone" placeholder="请输入手机号码" maxlength="11" class="modern-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户邮箱" prop="email">
              <el-input v-model="adminInfo.email" placeholder="请输入邮箱" maxlength="50" class="modern-input" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="用户名称" prop="username">
              <el-input v-model="adminInfo.username" placeholder="请输入用户名称" maxlength="30" class="modern-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户状态" prop="status">
              <el-radio-group v-model="adminInfo.status" class="modern-radio-group">
                <el-radio :label="1" class="modern-radio">正常</el-radio>
                <el-radio :label="2" class="modern-radio">停用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="用户岗位" prop="postId">
              <el-select placeholder="请选择岗位" v-model="adminInfo.postId" class="modern-select">
                <el-option v-for="item in postList" :key="item.id" :label="item.postName" :value="item.id"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户角色" prop="roleId">
              <el-select placeholder="请选择角色" v-model="adminInfo.roleId" class="modern-select">
                <el-option v-for="item in roleList" :key="item.id" :label="item.roleName" :value="item.id"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="24">
            <el-form-item label="个人简介" prop="note">
              <el-input v-model="adminInfo.note" type="textarea" placeholder="请输入内容" class="modern-textarea"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
        <span slot="footer" class="dialog-footer">
          <el-button type="primary" @click="editAdminInfo" class="modern-btn primary-btn">确 定</el-button>
          <el-button type="info" @click="editDialogVisible = false" class="modern-btn secondary-btn">取 消</el-button>
        </span>
      </el-form>
    </el-dialog>
  </div>
</template>



<script>
import Treeselect from 'vue3-treeselect'
import 'vue3-treeselect/dist/vue3-treeselect.css'
import {
  Search,
  Refresh,
  Plus,
  Edit,
  Delete,
  Key,
  Folder,
  FolderOpened
} from '@element-plus/icons-vue'

export default {
  components: {
    Treeselect
  },
  data() {
    return {
      expandedKeys: [], // 用于跟踪展开的节点key
      statusList: [{
        value: '1',
        label: '启用',
      }, {
        value: '2',
        label: '停用',
      }],
      Loading: false,
      queryParams: {},
      adminList: [],
      total: 0,
      addDialogVisible: false,
      deptList: [],// 确保初始化为空数组
      roleList: [],
      postList: [],
      addForm: {
        username: '',
        password: '',
        deptId: undefined,
        postId: undefined,
        roleId: undefined,
        email: '',
        nickname: '',
        status: 1,
        phone: '',
        note: ''
      },
      addFormRules: {
        deptId: [{required: true, message: '请选择部门', trigger: 'blur'}],
        postId: [{required: true, message: '请选择岗位', trigger: 'blur'}],
        roleId: [{required: true, message: '请选择角色', trigger: 'blur'}],
        username: [{required: true, message: '请输入用户账号', trigger: 'blur'}],
        password: [{required: true, message: '请输入用户密码', trigger: 'blur'}],
        status: [{required: true, message: '请选择状态', trigger: 'blur'}],
        email: [{required: true, message: '请输入用户邮箱', trigger: 'blur'}],
        nickname: [{required: true, message: '请输入用户昵称', trigger: 'blur'}],
        phone: [{required: true, message: '请输入用户手机', trigger: 'blur'}]
      },
      editDialogVisible: false,
      adminInfo: {},
      editFormRules: {
        deptId: [{required: true, message: '请选择部门', trigger: 'blur'}],
        postId: [{required: true, message: '请选择岗位', trigger: 'blur'}],
        roleId: [{required: true, message: '请选择角色', trigger: 'blur'}],
        username: [{required: true, message: '请输入用户账号', trigger: 'blur'}],
        status: [{required: true, message: '请选择状态', trigger: 'blur'}],
        email: [{required: true, message: '请输入用户邮箱', trigger: 'blur'}],
        nickname: [{required: true, message: '请输入用户昵称', trigger: 'blur'}],
        phone: [{required: true, message: '请输入用户手机', trigger: 'blur'}]
      },
      defaultProps: {
        children: 'children',
        label: 'label'
      },
      currentDeptId: null
    }
  },
  created() {
    this.getDeptVoList()
    this.getRoleVoList()
    this.getPostVoList()
    this.getAdminList()
  },
  methods: {
    // 表格行样式
    tableRowClassName({row, rowIndex}) {
      return 'modern-table-row';
    },
    
    // 查询列表
    async getAdminList() {
      this.Loading = true
      const {data: res} = await this.$api.queryAdminList(this.queryParams)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.adminList = res.data.list
        this.total = res.data.total
        this.Loading = false
      }
    },

    // 根据部门查询用户
    async loadAdminsByDept(deptId) {
      console.log('开始加载部门 ID:', deptId);
      this.Loading = true;
      const { data: res } = await this.$api.deptUsers(deptId);
      console.log('接口返回结果:', res);

      if (res.code !== 200) {
        this.$message.error(res.message);
      } else {
        if (!res.data || res.data.length === 0) {
          // 🔥 显式处理无数据的情况
          this.adminList = [];
          this.total = 0;
          this.queryParams.pageNum = 1;
          this.Loading = false;
          return;
        }

        // 添加字段映射逻辑
        const users = res.data.map(item => ({
          id: item.id,
          username: item.username,
          nickname: item.nickname,
          status: item.status,
          icon: item.icon,
          email: item.email,
          phone: item.phone,
          note: item.note,
          createTime: item.create_time || item.createTime,
          deptName: item.dept_name || item.deptName,
          postName: item.post_name || item.postName,
          roleName: item.role_name || item.roleName
        }));
        this.adminList = users;
        this.total = users.length;
        this.queryParams.pageNum = 1; // 回到第一页
      }
      this.Loading = false;
    }

    ,

    // 搜索按钮操作
    handleQuery() {
      this.getAdminList();
    },

    // 重置按钮操作
    resetQuery() {
      this.queryParams = {}
      this.getAdminList();
      this.$message.success("重置成功")
    },

    // pageSize
    handleSizeChange(newSize) {
      this.queryParams.pageSize = newSize
      this.getAdminList()
    },

    // pageNum
    handleCurrentChange(newPage) {
      this.queryParams.pageNum = newPage
      this.getAdminList()
    },

    // 修改用户状态
    async adminUpdateStatus(row) {
      let text = row.status === 2 ? "停用" : "启用";
      const confirmResult = await this.$confirm('确认要"' + text + '""' + row.username + '"用户吗?', "警告", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      }).catch(err => err);

      if (confirmResult !== 'confirm') {
        await this.getAdminList();
        return this.$message.info('已取消删除');
      }

      // 更新状态
      const { data: res } = await this.$api.updateAdminStatus(row.id, row.status);
      if (res.code === 200) {
        this.$message.success(text + "成功");

        // 手动更新当前行状态，避免重新加载整个列表
        const index = this.adminList.findIndex(item => item.id === row.id);
        if (index > -1) {
          this.$set(this.adminList, index, {...row}); // 强制触发 Vue 响应式更新
        }
      } else {
        this.$message.error(res.message);
      }
    },

    // 部门下拉列表
    async getDeptVoList() {
      const { data: res } = await this.$api.querySysDeptVoList()
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        const treeData = this.$handleTree.handleTree(res.data, "id")
        this.deptList = treeData
      }
    }
    ,

    // 角色下拉列表
    async getRoleVoList() {
      const {data: res} = await this.$api.querySysRoleVoList()
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.roleList = res.data
      }
    },

    // 岗位下拉列表
    async getPostVoList() {
      const {data: res} = await this.$api.querySysPostVoList()
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.postList = res.data
      }
    },

    // 监听添加用户对话框关闭
    addDialogClosed() {
      this.$refs.addFormRefForm.resetFields()
    },

    // 新增用户
    addAdmin() {
      this.$refs.addFormRefForm.validate(async valid => {
        if (!valid) return
        const {data: res} = await this.$api.addAdmin(this.addForm);
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.$message.success('新增用户成功')
          this.addDialogVisible = false
          await this.getAdminList()
        }
      })
    },

    // 展示编辑用户的对话框
    async showEditAdminDialog(id) {
      const {data: res} = await this.$api.adminInfo(id) // [adminInfo](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L333-L342)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.adminInfo = res.data
        
        // 如果用户有部门ID，获取部门信息以确保treeselect正确显示
        if (this.adminInfo.deptId) {
          try {
            const {data: deptRes} = await this.$api.deptInfo(this.adminInfo.deptId)
            if (deptRes.code === 200) {
              // 确保部门信息在treeselect选项中可用
              console.log('Department info loaded:', deptRes.data)
            }
          } catch (error) {
            console.warn('Failed to load department info:', error)
          }
        }
        
        this.editDialogVisible = true
      }
    },

    // 监听修改用户对话框的关闭事件
    editDialogClosed() {
      this.$refs.editFormRefForm.resetFields()
    },

    // 修改用户信息并提交
    editAdminInfo() {
      this.$refs.editFormRefForm.validate(async valid => {
        if (!valid) return
        const {data: res} = await this.$api.adminUpdate(this.adminInfo) // [adminUpdate](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L352-L361)
        if (res.code !== 200) {
          this.$message.error(res.message)
        } else {
          this.editDialogVisible = false
          await this.getAdminList()
          this.$message.success('修改用户成功')
        }
      })
    },

    // 删除
    async handleAdminDelete(row) {
      const confirmResult = await this.$confirm('是否确认删除用户为"' + row.username + '"的数据项？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).catch(err => err)
      if (confirmResult !== 'confirm') {
        return this.$message.info('已取消删除')
      }
      const {data: res} = await this.$api.deleteAdmin(row.id) // [deleteAdmin](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L373-L382)
      if (res.code !== 200) {
        this.$message.error(res.message)
      } else {
        this.$message.success('删除成功')
        await this.getAdminList()
      }
    },

    // 重置密码
    handleResetPwd(row) {
      this.$prompt('请输入"' + row.username + '"的新密码', "提示", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        closeOnClickModal: false,
        inputPattern: /^.{5,20}$/,
        inputErrorMessage: "用户密码长度必须介于 5 和 20 之间"
      }).then(({value}) => {
        this.$api.resetPassword(row.id, value).then(() => { // [resetPassword](file:///Users/apple/Desktop/zhangfan/2025-06-30/gin-web/src/api/index.js#L363-L371)
          this.$message.success("修改成功，新密码是：" + value);
        });
      }).catch(() => {
      });
    },

// 点击部门节点
    handleNodeExpand(data, node) {
      if (!this.expandedKeys.includes(node.key)) {
        this.expandedKeys.push(node.key)
      }
    },

    handleNodeCollapse(data, node) {
      this.expandedKeys = this.expandedKeys.filter(key => key !== node.key)
    },

    handleDeptClick(node, element) {
      let deptId;

      // 多种方式尝试获取 deptId
      if (element && element.data && element.data.id) {
        deptId = element.data.id;
      } else if (element && element.id) {
        deptId = element.id;
      } else if (node && node.key) {
        deptId = node.key;
      }
      console.log('最终获取到的 deptId:', deptId);
      if (!deptId) {
        this.$message.warning("无法获取部门ID");
        return;
      }
      this.currentDeptId = deptId;
      this.loadAdminsByDept(deptId);
    }





  }
}
</script>

<style scoped>
/* 主容器 - 渐变背景 */
.modern-admin-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
  position: relative;
}

/* 主标题样式 */
.main-title {
  font-size: 20px;
  font-weight: 600;
  color: #2c3e50;
  background: linear-gradient(45deg, #667eea, #764ba2);
   background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* 毛玻璃卡片基础样式 */
.glass-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-sizing: border-box;
  width: 100%;
}

.glass-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);
}

.main-card {
  padding: 24px;
}

/* 布局容器 */
.user-management-container {
  display: flex;
  gap: 20px;
  min-height: 600px;
  overflow: hidden;
}

.dept-tree-section {
  width: 280px;
  flex-shrink: 0;
  min-width: 200px;
}

.user-table-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
  overflow: hidden;
}

/* 卡片头部 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(103, 126, 234, 0.1);
}

.section-title {
  color: #2c3e50;
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  background: linear-gradient(45deg, #667eea, #764ba2);
   background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* 树形组件样式 */
.tree-card {
  padding: 20px;
}

.modern-tree {
  border: none;
  background: transparent;
}

.tree-node {
  color: #606266;
  font-size: 14px;
  transition: color 0.3s ease;
}

.tree-node:hover {
  color: #667eea;
}

.font-weight-bold {
  font-weight: 600;
}

/* 搜索表单样式 */
.search-card {
  padding: 20px;
  background: rgba(103, 126, 234, 0.05);
  border-radius: 12px;
  border: 1px solid rgba(103, 126, 234, 0.1);
  overflow: hidden;
}

.modern-form {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  width: 100%;
  box-sizing: border-box;
}

.modern-form .el-form-item {
  margin-bottom: 0;
  flex-shrink: 0;
}

.modern-form .el-form-item__label {
  color: #606266 !important;
  font-weight: 500;
}

.modern-form .el-form-item:last-child {
  display: flex;
  gap: 12px;
}

/* 现代化按钮 */
.modern-btn {
  border-radius: 8px;
  padding: 8px 20px;
  font-weight: 500;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.modern-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
}

.primary-btn {
  background: linear-gradient(45deg, #409EFF, #66B3FF);
  color: white;
}

.secondary-btn {
  background: linear-gradient(45deg, #909399, #B1B3B8);
  color: white;
}

.reset-btn {
  background: linear-gradient(45deg, #E6A23C, #EEBE77);
  color: white;
}

.success-btn {
  background: linear-gradient(45deg, #67C23A, #85CE61);
  color: white;
}

/* 输入框样式 */
.modern-input :deep(.el-input__wrapper),
.modern-select :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(103, 126, 234, 0.2);
  border-radius: 8px;
  box-shadow: none;
  transition: all 0.3s ease;
}

.modern-input :deep(.el-input__wrapper):hover,
.modern-select :deep(.el-input__wrapper):hover {
  border-color: #c0c4cc;
}

.modern-input :deep(.el-input__wrapper.is-focus),
.modern-select :deep(.el-input__wrapper.is-focus) {
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(103, 126, 234, 0.2);
  background: rgba(255, 255, 255, 1);
}

.modern-input :deep(.el-input__inner),
.modern-select :deep(.el-input__inner) {
  background: transparent;
  border: none;
  color: #2c3e50;
}

.modern-input :deep(.el-input__inner::placeholder),
.modern-select :deep(.el-input__inner::placeholder) {
  color: rgba(44, 62, 80, 0.6);
}

.modern-select :deep(.el-select-dropdown) {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.3);
}

.modern-date-picker {
  width: 190px;
  min-width: 150px;
}

/* 确保输入框不会超出容器 */
.modern-input,
.modern-select,
.modern-date-picker {
  max-width: 100%;
  box-sizing: border-box;
}

/* 表格样式 */
.table-card {
  padding: 20px;
  flex: 1;
  overflow: hidden;
  min-width: 0;
}

.modern-table {
  background: transparent;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  width: 100%;
}

.modern-table :deep(.el-table) {
  overflow-x: auto;
}

.modern-table :deep(.el-table__body-wrapper) {
  overflow-x: auto;
}

.modern-table :deep(.el-table__header) {
  background: linear-gradient(135deg, #667eea, #764ba2);
}

.modern-table :deep(.el-table__header th) {
  background: transparent !important;
  color: #2c3e50 !important;
  font-weight: 700 !important;
  border-bottom: none;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
  padding: 8px 12px !important;
  height: 40px;
}

.modern-table :deep(.el-table__header th .cell) {
  color: #2c3e50 !important;
  font-weight: 700 !important;
  text-shadow: 0 1px 2px rgba(255, 255, 255, 0.8);
}

.modern-table :deep(.el-table__row) {
  transition: all 0.3s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: rgba(103, 126, 234, 0.05) !important;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.modern-table :deep(.el-table__row td) {
  border: none;
  color: #2c3e50;
  padding: 8px 12px !important;
  height: 40px;
}

.modern-table :deep(.el-table__cell) {
  padding: 8px 12px !important;
}

.modern-table :deep(.el-table__empty-block) {
  background: rgba(255, 255, 255, 0.05);
}

.modern-table :deep(.el-table__empty-text) {
  color: rgba(44, 62, 80, 0.6);
}

/* 表格内组件 */
.modern-tag {
  border-radius: 6px;
  font-weight: 500;
}

.modern-avatar {
  border: 2px solid rgba(255, 255, 255, 0.3);
  transition: transform 0.3s ease;
}

.modern-avatar:hover {
  transform: scale(1.1);
}

.modern-switch :deep(.el-switch__core) {
  border-radius: 12px;
}

.modern-text {
  color: #2c3e50;
}

/* 操作按钮 */
.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.operation-buttons .el-button {
  transition: all 0.3s ease;
}

.operation-buttons .el-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

/* 分页样式 */
.pagination-card {
  padding: 20px;
  text-align: center;
}

.modern-pagination :deep(.el-pagination) {
  justify-content: center;
}

.modern-pagination :deep(.el-pagination .el-pager li),
.modern-pagination :deep(.el-pagination .btn-prev),
.modern-pagination :deep(.el-pagination .btn-next),
.modern-pagination :deep(.el-pagination .el-pagination__sizes),
.modern-pagination :deep(.el-pagination .el-pagination__jump) {
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #ffffff;
  border-radius: 6px;
  margin: 0 2px;
}

.modern-pagination :deep(.el-pagination .el-pager li:hover),
.modern-pagination :deep(.el-pagination .btn-prev:hover),
.modern-pagination :deep(.el-pagination .btn-next:hover) {
  background: rgba(255, 255, 255, 0.2);
  color: #ffffff;
}

.modern-pagination :deep(.el-pagination .el-pager li.active) {
  background: linear-gradient(45deg, #409EFF, #66B3FF);
  color: #ffffff;
}

/* 对话框样式 */
.modern-dialog :deep(.el-dialog) {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.2);
}

.modern-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px 16px 0 0;
  padding: 16px 20px;
}

.modern-dialog :deep(.el-dialog__title) {
  color: #ffffff;
  font-weight: 600;
  font-size: 16px;
}

.modern-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #ffffff;
  font-size: 16px;
}

.modern-dialog-form {
  padding: 20px;
}

.modern-dialog-form .el-form-item__label {
  color: #606266 !important;
  font-weight: 500;
}

.modern-dialog-form .modern-input :deep(.el-input__wrapper),
.modern-dialog-form .modern-select :deep(.el-input__wrapper) {
  background: rgba(0, 0, 0, 0.02);
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: none;
  transition: all 0.3s ease;
}

.modern-dialog-form .modern-input :deep(.el-input__inner),
.modern-dialog-form .modern-select :deep(.el-input__inner) {
  background: transparent;
  border: none;
  color: #333333;
}

.modern-dialog-form .modern-textarea :deep(.el-textarea__inner) {
  background: rgba(0, 0, 0, 0.02);
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  color: #333333;
  transition: all 0.3s ease;
}

.modern-dialog-form .modern-input :deep(.el-input__wrapper):hover,
.modern-dialog-form .modern-select :deep(.el-input__wrapper):hover {
  border-color: #c0c4cc;
}

.modern-dialog-form .modern-input :deep(.el-input__wrapper.is-focus),
.modern-dialog-form .modern-select :deep(.el-input__wrapper.is-focus),
.modern-dialog-form .modern-textarea :deep(.el-textarea__inner):focus {
  border-color: #409EFF;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);
  background: rgba(255, 255, 255, 1);
}

.modern-dialog-form .modern-select {
  width: 100%;
}

.modern-radio-group .modern-radio {
  margin-right: 20px;
}

.modern-dialog-form .modern-radio :deep(.el-radio__label) {
  color: #606266 !important;
}

.modern-dialog-form .modern-radio :deep(.el-radio__input.is-checked .el-radio__inner) {
  border-color: #409EFF;
  background: #409EFF;
}

.modern-treeselect :deep(.vue-treeselect__control) {
  background: rgba(0, 0, 0, 0.02);
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: none;
  transition: all 0.3s ease;
}

.modern-treeselect :deep(.vue-treeselect__control):hover {
  border-color: #c0c4cc;
}

.modern-treeselect :deep(.vue-treeselect__control--is-focused) {
  border-color: #409EFF;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);
  background: rgba(255, 255, 255, 1);
}

.modern-treeselect :deep(.vue-treeselect__placeholder),
.modern-treeselect :deep(.vue-treeselect__single-value),
.modern-treeselect :deep(.vue-treeselect__option) {
  color: #606266 !important;
}

.dialog-footer {
  text-align: center;
  padding: 20px;
  border-top: 1px solid #f0f0f0;
}

/* 响应式布局 */
@media (max-width: 1200px) {
  .user-management-container {
    flex-direction: column;
    gap: 16px;
  }
  
  .dept-tree-section {
    width: 100%;
    order: 2;
  }
  
  .user-table-section {
    order: 1;
  }
  
  .modern-form {
    flex-direction: column;
    align-items: stretch;
  }
  
  .modern-form .el-form-item {
    width: 100%;
  }
  
  .button-group {
    flex-direction: column;
    width: 100%;
  }
  
  .button-group .modern-btn {
    width: 100%;
  }
}

@media (max-width: 768px) {
  .modern-admin-container {
    padding: 10px;
  }
  
  .main-card {
    padding: 16px;
  }
  
  .search-card, .table-card, .tree-card {
    padding: 12px;
  }
  
  .modern-form {
    gap: 8px;
  }
  
  .modern-input :deep(.el-input__inner),
  .modern-select :deep(.el-input__inner),
  .modern-date-picker :deep(.el-input__inner) {
    font-size: 14px;
  }
  
  .modern-table :deep(.el-table) {
    font-size: 12px;
  }
  
  .main-title {
    font-size: 18px;
  }
  
  .section-title {
    font-size: 14px;
  }
}

@media (max-width: 480px) {
  .modern-admin-container {
    padding: 8px;
  }
  
  .main-card {
    padding: 12px;
  }
  
  .search-card, .table-card, .tree-card {
    padding: 8px;
  }
  
  .user-management-container {
    gap: 12px;
  }
  
  .main-title {
    font-size: 16px;
  }
  
  .modern-form .el-form-item__label {
    font-size: 12px !important;
  }
  
  .modern-table :deep(.el-table) {
    font-size: 11px;
  }
}
</style>
