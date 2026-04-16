import request from '@/utils/request'

export const getApprovalRecords = (contractId) => {
  return request({
    url: `/contracts/${contractId}/approvals`,
    method: 'get'
  })
}

export const getPendingApprovals = () => {
  return request({
    url: '/workflow/pending',
    method: 'get'
  })
}

export const createWorkflow = (data) => {
  return request({
    url: '/workflow/create',
    method: 'post',
    data
  })
}

export const approveWorkflow = (data) => {
  return request({
    url: '/workflow/approve',
    method: 'post',
    data
  })
}

export const rejectWorkflow = (data) => {
  return request({
    url: '/workflow/reject',
    method: 'post',
    data
  })
}

export const createApproval = (data) => {
  return request({
    url: `/contracts/${data.contract_id}/approvals`,
    method: 'post',
    data
  })
}

export const updateApproval = (id, data) => {
  return request({
    url: `/approvals/${id}`,
    method: 'put',
    data
  })
}

export const getReminders = (contractId) => {
  return request({
    url: `/contracts/${contractId}/reminders`,
    method: 'get'
  })
}

export const createReminder = (data) => {
  return request({
    url: `/contracts/${data.contract_id}/reminders`,
    method: 'post',
    data
  })
}

export const sendReminder = (id) => {
  return request({
    url: `/reminders/${id}/send`,
    method: 'post'
  })
}

export const getExpiringContracts = (days = 30) => {
  return request({
    url: '/expiring-contracts',
    method: 'get',
    params: { days }
  })
}

export const getStatistics = () => {
  return request({
    url: '/statistics',
    method: 'get'
  })
}

export const getNotificationCounts = () => {
  return request({
    url: '/notifications/count',
    method: 'get'
  })
}

export const getWorkflowStatus = (contractId) => {
  return request({
    url: `/workflow/${contractId}/status`,
    method: 'get'
  })
}

export const sendApprovalReminder = (contractId) => {
  return request({
    url: `/workflow/${contractId}/remind`,
    method: 'post'
  })
}

export const getMyNotifications = () => {
  return request({
    url: '/notifications',
    method: 'get'
  })
}

export const markNotificationRead = (id) => {
  return request({
    url: `/notifications/${id}/read`,
    method: 'put'
  })
}

export const getUnreadNotificationCount = () => {
  return request({
    url: '/notifications/unread-count',
    method: 'get'
  })
}

export const deleteNotification = (id) => {
  return request({
    url: `/notifications/${id}`,
    method: 'delete'
  })
}

export const deleteAllNotifications = () => {
  return request({
    url: '/notifications/all',
    method: 'delete'
  })
}