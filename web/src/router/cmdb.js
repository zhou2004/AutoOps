import Host from '@/views/cmdb/cmdbHost.vue'
import Group from '@/views/cmdb/cmdbGroup.vue'
import Db from '@/views/cmdb/cmdbDB.vue'
import SSH from '@/views/cmdb/Host/SSH.vue'
import DBdetails from '@/views/cmdb/DBdetails.vue'
import PhysicalMachine from '@/views/cmdb/physicalMachine.vue'
import NetworkDevice from '@/views/cmdb/networkDevice.vue'
import AssetPermission from '@/views/cmdb/assetPermission.vue'
import CmdbUserGroup from '@/views/cmdb/cmdbUserGroup.vue'
import CredentialPermission from '@/views/cmdb/credentialPermission.vue'
import MyAssetPermissions from '@/views/cmdb/myAssetPermissions.vue'

const routes = [
    {
        path: '/cmdb/ecs',
        component: Host,
        meta: { sTitle: '资产管理', tTitle: '主机管理' }
    },
    {
        path: '/cmdb/group',
        component: Group,
        meta: { sTitle: '资产管理', tTitle: '业务分组' }
    },
    {
        path: '/cmdb/db',
        component: Db,
        meta: { sTitle: '资产管理', tTitle: '数据管理' }
    },
    {
        path: '/cmdb/ssh',
        component: SSH,
        meta: { sTitle: '资产管理', tTitle: '终端登录' }
    },
    {
        path: '/cmdb/dbdetails',
        component: DBdetails,
        meta: { sTitle: '数据管理', tTitle: '数据库操作' }
    },
    {
        path: '/cmdb/physical',
        component: PhysicalMachine,
        meta: { sTitle: '资产管理', tTitle: '物理机管理' }
    },
    {
        path: '/cmdb/network',
        component: NetworkDevice,
        meta: { sTitle: '资产管理', tTitle: '网络设备管理' }
    },
    {
        path: '/cmdb/asset-permission',
        component: AssetPermission,
        meta: { sTitle: '资产管理', tTitle: '资产授权' }
    },
    {
        path: '/cmdb/user-group',
        component: CmdbUserGroup,
        meta: { sTitle: '资产管理', tTitle: '用户组管理' }
    },
    {
        path: '/cmdb/credential-permission',
        component: CredentialPermission,
        meta: { sTitle: '资产管理', tTitle: '凭据授权' }
    },
    {
        path: '/cmdb/my-assets',
        component: MyAssetPermissions,
        meta: { sTitle: '资产管理', tTitle: '我的授权资产' }
    }
]

export default routes
