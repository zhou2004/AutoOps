import App from '@/views/app/application.vue'
import QuickRelease from '@/views/app/app_quick_release.vue'
import QuickTemp from '@/views/app/app_quick_temp.vue'

const routes = [
    {
        path: '/app/application',
        component: App,
        meta: {sTitle: '服务管理', tTitle: '应用列表'}
    },
    {
        path: '/app/quick-release',
        component: QuickRelease,
        meta: {sTitle: '服务管理', tTitle: '快速发布'}
    },
    {
        path: '/app/quick-temp/:id',
        component: QuickTemp,
        meta: {sTitle: '服务管理', tTitle: '发布模板'}
    },
]

export default routes
