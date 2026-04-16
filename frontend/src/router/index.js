import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/store/user'

// 总监角色列表
const directorRoles = ['sales_director', 'tech_director', 'finance_director']

// 总监可见的路由
const directorAllowedRoutes = ['dashboard', 'contracts', 'contracts/:id', 'customers', 'approvals', 'reminders']

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/Register.vue'),
    meta: { title: '注册' }
  },
  {
    path: '/',
    component: () => import('@/views/Layout.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '仪表盘', icon: 'DataAnalysis', permission: 'dashboard' }
      },
      {
        path: 'contracts',
        name: 'Contracts',
        component: () => import('@/views/Contract.vue'),
        meta: { title: '合同管理', icon: 'Document', permission: 'contract.read' }
      },
      {
        path: 'contracts/:id',
        name: 'ContractDetail',
        component: () => import('@/views/ContractDetail.vue'),
        meta: { title: '合同详情', hidden: true, permission: 'contract.read' }
      },
      {
        path: 'customers',
        name: 'Customers',
        component: () => import('@/views/Customer.vue'),
        meta: { title: '客户管理', icon: 'User', permission: 'customer.read' }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/User.vue'),
        meta: { title: '用户管理', icon: 'UserFilled', permission: 'user.manage' }
      },
      {
        path: 'approvals',
        name: 'Approvals',
        component: () => import('@/views/Approval.vue'),
        meta: { title: '审批管理', icon: 'Check', permission: 'approval.view' }
      },
      {
        path: 'reminders',
        name: 'Reminders',
        component: () => import('@/views/Reminder.vue'),
        meta: { title: '到期提醒', icon: 'Bell', permission: 'dashboard' }
      },
      {
        path: 'audit',
        name: 'Audit',
        component: () => import('@/views/Audit.vue'),
        meta: { title: '审计日志', icon: 'Document', permission: 'audit.view' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

router.beforeEach((to, from, next) => {
  try {
    const userStore = useUserStore()
    
    if (to.meta.requiresAuth && !userStore.token) {
      next('/login')
      return
    }
    
    if ((to.path === '/login' || to.path === '/register') && userStore.token) {
      next('/dashboard')
      return
    }
    
    // 获取用户角色
    const userRole = userStore.userInfo?.role || ''
    const isDirector = directorRoles.includes(userRole)
    
    // 总监角色路由限制
    if (isDirector) {
      const path = to.path.replace('/', '')
      // 检查是否匹配允许的路由（支持动态路由参数）
      const pathMatches = directorAllowedRoutes.some(allowed => {
        if (allowed.includes(':')) {
          // 将动态路由参数替换成正则匹配
          const pattern = allowed.replace(/:[^/]+/g, '[^/]+')
          const regex = new RegExp(`^${pattern}$`)
          return regex.test(path)
        }
        return path === allowed
      })
      if (!pathMatches && path !== '') {
        next('/dashboard')
        return
      }
    }
    
    if (to.meta.permission) {
      const hasPermission = userStore.hasPermission(to.meta.permission)
      if (!hasPermission) {
        next('/dashboard')
        return
      }
    }
  } catch (error) {
    console.error('Router error:', error)
  }
  
  next()
})

export default router
