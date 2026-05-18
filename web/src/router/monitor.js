import AlarmNotify from '@/views/monitor/Alarm-notify.vue'
import AlarmRules from '@/views/monitor/Alarm-rules.vue'
import AlarmHistory from '@/views/monitor/alarm-history.vue'
import DataSource from '@/views/monitor/DataSource.vue'
import DBLog from '@/views/monitor/DBLog.vue'
import LoginLog from '@/views/monitor/LoginLog.vue'
import OperatorLog from '@/views/monitor/Operator.vue'
import MonitorBase from '@/views/monitor/base.vue'
import HttpsMonitor from '@/views/monitor/https.vue'
import DomainCert from '@/views/monitor/DomainCert.vue'
import APIEndpoint from '@/views/monitor/APIEndpoint.vue'
import Incident from '@/views/monitor/Incident.vue'

const routes = [
    {
        path: '/monitor/alarm/notify',
        component: AlarmNotify,
        meta: { sTitle: '监控中心', tTitle: '告警通知' }
    },
    {
        path: '/monitor/alarm/rules',
        component: AlarmRules,
        meta: { sTitle: '监控中心', tTitle: '告警配置' }
    },
    {
        path: '/monitor/datasource',
        component: DataSource,
        meta: { sTitle: '监控中心', tTitle: '数据源管理' }
    },
    {
        path: '/monitor/alarm/history',
        component: AlarmHistory,
        meta: { sTitle: '监控中心', tTitle: '告警历史' }
    },
    {
        path: '/monitor/log/db',
        component: DBLog,
        meta: { sTitle: '审计日志', tTitle: '数据库日志' }
    },
    {
        path: '/monitor/log/login',
        component: LoginLog,
        meta: { sTitle: '审计日志', tTitle: '登录日志' }
    },
    {
        path: '/monitor/log/operator',
        component: OperatorLog,
        meta: { sTitle: '审计日志', tTitle: '操作日志' }
    },
    {
        path: '/monitor/base',
        component: MonitorBase,
        meta: { sTitle: '基础监控', tTitle: '基础设施监控' }
    },
    {
        path: '/monitor/https',
        component: HttpsMonitor,
        meta: { sTitle: '基础监控', tTitle: 'HTTPS监控' }
    },
    {
        path: '/monitor/domain-cert',
        component: DomainCert,
        meta: { sTitle: '基础监控', tTitle: '域名证书监控' }
    },
    {
        path: '/monitor/api-endpoint',
        component: APIEndpoint,
        meta: { sTitle: '基础监控', tTitle: 'API端点监控' }
    },
    {
        path: '/monitor/incident',
        component: Incident,
        meta: { sTitle: '监控中心', tTitle: '故障管理' }
    },
]

export default routes
