<template>
  <div class="db-details-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>数据库详情</span>
          <el-button 
            type="primary" 
            size="small" 
            @click="getDBDetails"
            style="margin-left: 10px;"
          >
            <el-icon><Refresh /></el-icon>
            <span>刷新</span>
          </el-button>
        </div>
      </template>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="数据库名称">{{ dbInfo.name }}</el-descriptions-item>
      <el-descriptions-item label="数据库类型">
        <el-tag :type="dbInfo.type === 1 ? 'success' : (dbInfo.type === 2 ? 'warning' : 'danger')">
          {{ dbInfo.type === 1 ? 'MySQL' : (dbInfo.type === 2 ? 'PostgreSQL' : 'Redis') }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="所属账号">{{ dbInfo.accountAlias }}</el-descriptions-item>
      <el-descriptions-item label="业务分组">{{ dbInfo.groupName }}</el-descriptions-item>
      <el-descriptions-item label="标签">{{ dbInfo.tags }}</el-descriptions-item>
      <el-descriptions-item label="描述">{{ dbInfo.description }}</el-descriptions-item>
      <el-descriptions-item label="创建时间">{{ formatDate(dbInfo.createdAt) }}</el-descriptions-item>
      <el-descriptions-item label="更新时间">{{ formatDate(dbInfo.updatedAt) }}</el-descriptions-item>
    </el-descriptions>
    </el-card>

    <div class="sql-execute-container">
      <el-card>
        <template #header>
          <div class="card-header">
            <span>SQL执行</span>
          </div>
        </template>

        <div class="sql-form-container">
          <el-form :model="sqlForm" label-width="100px">
            <el-form-item label="数据库名称">
              <el-input v-model="dbInfo.name" disabled />
            </el-form-item>
            <el-form-item label="选择数据库">
              <el-select 
                v-model="sqlForm.databaseName" 
                placeholder="请选择数据库"
                @focus="getDatabaseList"
              >
                <el-option 
                  v-for="db in databaseList" 
                  :key="db" 
                  :label="db" 
                  :value="db" 
                />
              </el-select>
              <span v-if="dbConnectionInfo && dbConnectionInfo.host" style="margin-left: 10px;">
              </span>
            </el-form-item>
            <el-form-item label="SQL类型">
              <el-select 
                v-model="sqlForm.sqlType" 
                placeholder="请选择SQL类型"
                @change="updateSQLPlaceholder"
              >
                <el-option label="查询" value="select">
                  <span style="float: left">查询</span>
                  <span style="float: right; color: #8492a6; font-size: 13px">select id,name from 表名;</span>
                </el-option>
                <el-option label="插入" value="insert">
                  <span style="float: left">插入</span>
                  <span style="float: right; color: #8492a6; font-size: 13px">INSERT INTO `表名` (`字段1`, `字段2`) VALUES ('值1', '值2');</span>
                </el-option>
                <el-option label="更新" value="update">
                  <span style="float: left">更新</span>
                  <span style="float: right; color: #8492a6; font-size: 13px">UPDATE `表名` SET `字段1`='新值1' WHERE `id`=1;</span>
                </el-option>
                <el-option label="删除" value="delete">
                  <span style="float: left">删除</span>
                  <span style="float: right; color: #8492a6; font-size: 13px">DELETE FROM `表名` WHERE `id`=1;</span>
                </el-option>
                <el-option label="原生SQL" value="raw" />
              </el-select>
            </el-form-item>
            <el-form-item label="SQL语句">
              <el-input
                v-model="sqlForm.sql"
                type="textarea"
                :rows="5"
                :placeholder="sqlPlaceholder"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="executeSQL">执行</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-card>

      <el-card>
        <template #header>
          <div class="card-header">
            <span>执行结果</span>
          </div>
        </template>
        <div class="result-container">
          <pre>{{ executionResult }}</pre>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { useRoute } from 'vue-router'
import cmdbAPI from '@/api/cmdb'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

export default {
  setup() {
    const route = useRoute()
    const dbInfo = ref({
      name: '',
      type: undefined,
      accountAlias: '',
      groupName: '',
      tags: '',
      description: '',
      createdAt: '',
      updatedAt: ''
    })

    const sqlForm = ref({
      sqlType: 'select',
      sql: '',
      databaseName: ''
    })

    const executionResult = ref(`-- 查询（SELECT）
SELECT * FROM \`表名\` WHERE \`字段1\` = '值1';
SELECT * FROM \`表名\` WHERE \`字段1\` = '值1' AND \`字段2\` = '值2';
SELECT * FROM \`表名\` WHERE \`字段1\` = '值1' OR \`字段2\` = '值2';
SELECT * FROM \`表名\` WHERE \`字段\` LIKE '%关键词%';
SELECT * FROM \`表名\` WHERE \`id\` IN (1, 2, 3);
-- 插入（INSERT
INSERT INTO \`表名\` (\`字段1\`, \`字段2\`, \`字段3\`) VALUES ('值1', '值2', '值3');
INSERT INTO \`表名\` (\`字段1\`, \`字段2\`) VALUES ('值1a', '值2a'), ('值1b', '值2b'), ('值1c', '值2c');
-- 修改（UPDATE）
UPDATE \`表名\` SET \`字段1\` = '新值1' WHERE \`id\` = 1;
UPDATE \`表名\` SET \`字段1\` = '新值1' WHERE \`字段A\` = '条件A' AND \`字段B\` = '条件B';
UPDATE \`表名\` SET \`字段1\` = '新值1', \`字段2\` = '新值2' WHERE \`id\` = 1;
-- 删除（DELETE）
DELETE FROM \`表名\` WHERE \`id\` = 1;
DELETE FROM \`表名\` WHERE \`字段1\` = '值1' AND \`字段2\` = '值2';
DELETE FROM \`表名\` WHERE \`id\` IN (1, 2, 3, 4, 5);`)
    const databaseList = ref([])
    const dbConnectionInfo = ref({
      host: '',
      port: ''
    })
    const isLoadingDatabases = ref(false)
    const sqlPlaceholder = ref('请输入SQL语句')

    const updateSQLPlaceholder = (type) => {
      switch(type) {
        case 'select':
          sqlPlaceholder.value = '示例: SELECT * FROM `表名` WHERE `字段1` = 值1;'
          break
        case 'insert':
          sqlPlaceholder.value = '示例: INSERT INTO `表名` (`字段1`, `字段2`) VALUES (\'值1\', \'值2\');'
          break
        case 'update':
          sqlPlaceholder.value = '示例: UPDATE `表名` SET `字段1`=\'新值1\' WHERE `id`=1;'
          break
        case 'delete':
          sqlPlaceholder.value = '示例: DELETE FROM `表名` WHERE `id`=1;'
          break
        default:
          sqlPlaceholder.value = '请输入SQL语句'
      }
    }

    const getDatabaseList = async () => {
      try {
        const id = route.params.id || route.query.id
        if (!id) {
          ElMessage.error('数据库ID不能为空')
          return
        }

        isLoadingDatabases.value = true
        const res = await cmdbAPI.executeDatabase({ databaseId: Number(id) })
        
        if (res.data?.code === 200 && res.data.data) {
          databaseList.value = res.data.data.databases || []
          dbConnectionInfo.value = {
            host: res.data.data.host || '',
            port: res.data.data.port || ''
          }
          if (databaseList.value.length > 0) {
            ElMessage.success(`成功获取 ${databaseList.value.length} 个数据库`)
          } else {
            ElMessage.warning('该数据库实例中没有数据库')
          }
        } else {
          ElMessage.error(res.data?.message || '获取数据库列表失败')
        }
      } catch (error) {
        console.error('获取数据库列表失败:', error)
        ElMessage.error('获取数据库列表失败: ' + error.message)
      } finally {
        isLoadingDatabases.value = false
      }
    }

    const getDBDetails = async () => {
      try {
        const id = route.params.id || route.query.id
        console.log('从路由获取的ID:', id)
        
        if (!id) {
          ElMessage.error('数据库ID不能为空')
          return
        }

        // 直接使用request工具构造请求
        const res = await request({
          url: 'cmdb/database/info',
          method: 'get',
          params: { id: String(id) }
        })
        
        if (res.data?.code === 200) {
          const data = res.data.data
          dbInfo.value = {
            name: data.name,
            type: data.type,
            accountAlias: `账号ID: ${data.accountId}`,
            groupName: `分组ID: ${data.groupId}`,
            tags: data.tags,
            description: data.description,
            createdAt: data.createdAt,
            updatedAt: data.updatedAt
          }
        } else {
          ElMessage.error(res.data?.message || '获取数据库详情失败')
        }
      } catch (error) {
        console.error('获取数据库详情失败:', error)
        ElMessage.error('获取数据库详情失败: ' + error.message)
      }
    }

    const executeSQL = async () => {
      if (!sqlForm.value.sql) {
        ElMessage.warning('请输入SQL语句')
        return
      }
      if (!sqlForm.value.databaseName) {
        ElMessage.warning('请选择数据库')
        return
      }

      const requestData = {
        databaseId: Number(route.params.id || route.query.id),
        databaseName: sqlForm.value.databaseName,
        sql: sqlForm.value.sql
      }

      console.log('准备执行SQL，请求数据:', JSON.stringify(requestData, null, 2))
      console.log('SQL类型:', sqlForm.value.sqlType)

      try {
        let res
        switch (sqlForm.value.sqlType) {
          case 'select':
            console.log('调用executeSelectSQL')
            res = await cmdbAPI.executeSelectSQL(requestData)
            break
          case 'insert':
            console.log('调用executeInsertSQL')
            res = await cmdbAPI.executeInsertSQL(requestData)
            break
          case 'update':
            console.log('调用executeUpdateSQL')
            res = await cmdbAPI.executeUpdateSQL(requestData)
            break
          case 'delete':
            console.log('调用executeDeleteSQL')
            res = await cmdbAPI.executeDeleteSQL(requestData)
            break
          case 'raw':
            console.log('调用executeRawSQL')
            res = await cmdbAPI.executeRawSQL(requestData)
            break
          default:
            ElMessage.warning('请选择有效的SQL类型')
            return
        }

        console.log('API响应:', JSON.stringify(res?.data, null, 2))

        if (res.data?.code === 200) {
          executionResult.value = JSON.stringify(res.data.data, null, 2)
          ElMessage.success('执行成功')
        } else {
          executionResult.value = JSON.stringify(res.data, null, 2)
          ElMessage.error(res.data?.message || '执行失败')
        }
      } catch (error) {
        console.error('执行SQL失败:', error)
        executionResult.value = error.message
        ElMessage.error('执行失败: ' + error.message)
      }
    }

    const formatDate = (dateString) => {
      if (!dateString) return ''
      const date = new Date(dateString)
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    }

    onMounted(() => {
      getDBDetails()
    })

    return {
      dbInfo,
      sqlForm,
      executionResult,
      databaseList,
      dbConnectionInfo,
      isLoadingDatabases,
      executeSQL,
      formatDate,
      getDatabaseList
    }
  }
}
</script>

<style scoped>
.db-details-container {
  padding: 20px;
}

.card-header {
  font-size: 18px;
  font-weight: bold;
}

.sql-execute-container {
  margin-top: 20px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.sql-form-container {
  padding: 10px;
}

.result-container {
  padding: 10px;
  min-height: 300px;
  max-height: 500px;
  overflow: auto;
  background-color: #000;
  border-radius: 4px;
  color: #fff;
  font-family: monospace;
}

pre {
  white-space: pre-wrap;
  word-wrap: break-word;
  margin: 0;
  color: #fff;
}
</style>
