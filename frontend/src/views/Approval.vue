<template>
  <div class="approval-page">
    <!-- 审批统计卡片 - 放在最上面 -->
    <div class="approval-stats" v-if="isAdmin" style="margin-bottom: 16px;">
      <el-row :gutter="12">
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon pending"><el-icon><Clock /></el-icon></div>
              <div class="stat-info">
                <div class="stat-value">{{ stats.pendingCount }}</div>
                <div class="stat-label">待审批</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon sales"><el-icon><User /></el-icon></div>
              <div class="stat-info">
                <div class="stat-value">{{ stats.salesPending }}</div>
                <div class="stat-label">销售总监</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon tech"><el-icon><Monitor /></el-icon></div>
              <div class="stat-info">
                <div class="stat-value">{{ stats.techPending }}</div>
                <div class="stat-label">技术总监</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon finance"><el-icon><Money /></el-icon></div>
              <div class="stat-info">
                <div class="stat-value">{{ stats.financePending }}</div>
                <div class="stat-label">财务总监</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <el-card class="approval-card">
      <template #header>
        <div class="header">
          <div class="header-left">
            <el-icon class="header-icon"><Check /></el-icon>
            <span class="header-title">待审批列表</span>
            <el-tag type="warning" effect="dark" style="margin-left: 12px;">{{ tableData.length }} 条</el-tag>
</div>
  </div>
</template>

<!-- 审批流程步骤条 -->
<div class="flow-steps" v-if="tableData.length > 0" style="margin-bottom: 16px;">
  <el-steps :active="currentStepIndex" finish-status="success" align-center>
    <el-step title="销售提交" />
    <el-step title="销售总监" />
    <el-step title="技术总监" />
    <el-step title="财务总监" />
    <el-step title="完成生效" />
  </el-steps>
</div>

<el-table :data="tableData" style="width: 100%" v-loading="loading" class="approval-table" :cell-style="{ padding: '8px 0' }">
  <el-table-column prop="contract_no" label="合同编号" min-width="130">
    <template #default="{ row }">
      <div class="contract-no">
        <el-icon><Document /></el-icon>
        {{ row.contract_no }}
      </div>
    </template>
  </el-table-column>
  <el-table-column prop="title" label="合同标题" min-width="180">
    <template #default="{ row }">
      <div class="contract-title">{{ row.title }}</div>
    </template>
  </el-table-column>
  <el-table-column prop="creator" label="销售人员" min-width="100">
    <template #default="{ row }">
      <span v-if="row.creator">{{ row.creator.full_name || row.creator.username }}</span>
      <span v-else>-</span>
    </template>
  </el-table-column>
  <el-table-column prop="amount" label="金额" min-width="110">
    <template #default="{ row }">
      <span class="amount">¥{{ row.amount?.toLocaleString() }}</span>
    </template>
  </el-table-column>
  <el-table-column prop="level" label="当前审批级别" min-width="120">
    <template #default="{ row }">
      <div class="level-info">
        <el-tag :type="getLevelType(row.level)" effect="dark" round size="small">
          {{ row.level_name }}
        </el-tag>
      </div>
    </template>
  </el-table-column>
  <el-table-column prop="status" label="状态" min-width="90">
    <template #default="{ row }">
      <el-tag :type="getStatusTagType(row)" effect="dark" round size="small">
        {{ getStatusDisplay(row) }}
      </el-tag>
    </template>
  </el-table-column>
  <el-table-column prop="created_at" label="提交时间" min-width="140">
    <template #default="{ row }">
      <span class="time">{{ formatDateTime(row.created_at) }}</span>
    </template>
  </el-table-column>
  <el-table-column label="操作" width="140" fixed="right">
    <template #default="{ row }">
      <div class="action-buttons">
        <el-tooltip content="查看详情" placement="top">
          <el-button type="primary" link @click="handleView(row)">
            <el-icon><View /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="审批通过" placement="top" v-if="row.status === 'pending'">
          <el-button type="success" link @click="handleApprove(row)" class="approve-btn">
            <el-icon><Check /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="审批拒绝" placement="top" v-if="row.status === 'pending'">
          <el-button type="danger" link @click="handleReject(row)" class="reject-btn">
            <el-icon><Close /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </template>
</el-table-column>
</el-table>
</el-card>

    <el-card style="margin-top: 20px" v-if="isAdmin">
      <template #header>
        <div class="header">
          <span>状态变更审批</span>
          <el-tag type="warning">{{ statusChangeData.length }} 条待审批</el-tag>
        </div>
      </template>
      <el-table :data="statusChangeData" style="width: 100%" v-loading="statusChangeLoading">
        <el-table-column prop="contract.contract_no" label="合同编号" width="150" />
        <el-table-column prop="contract.title" label="合同标题" />
        <el-table-column prop="from_status" label="原状态" width="100">
          <template #default="{ row }">
            <el-tag>{{ getStatusText(row.from_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="to_status" label="目标状态" width="100">
          <template #default="{ row }">
            <el-tag type="warning">{{ getStatusText(row.to_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="申请原因" />
        <el-table-column prop="requester.full_name" label="申请人" width="100" />
        <el-table-column prop="created_at" label="申请时间" width="160">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
<el-table-column label="操作" width="140" fixed="right">
  <template #default="{ row }">
    <div class="action-buttons">
<el-tooltip content="查看" placement="top">
        <el-button type="primary" link @click="handleViewStatusChange(row)">
          <el-icon><View /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="通过" placement="top">
        <el-button type="success" link @click="handleApproveStatusChange(row)">
          <el-icon><Check /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="拒绝" placement="top">
        <el-button type="danger" link @click="handleRejectStatusChange(row)">
          <el-icon><Close /></el-icon>
        </el-button>
      </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="合同审批" width="600px">
      <el-form ref="formRef" :model="formData" label-width="100px">
        <el-form-item label="合同编号">
          <el-input v-model="currentContract.contract_no" disabled />
        </el-form-item>
        <el-form-item label="合同标题">
          <el-input v-model="currentContract.title" disabled />
        </el-form-item>
        <el-form-item label="合同金额">
          <el-input :value="'¥' + currentContract.amount?.toFixed(2)" disabled />
        </el-form-item>
        <el-form-item label="审批级别">
          <el-tag :type="getLevelType(formData.level)">
            {{ getApprovalButtonText(formData.level) }}
          </el-tag>
        </el-form-item>
        <el-form-item label="审批结果" prop="status">
          <el-radio-group v-model="formData.status">
            <el-radio label="approved">通过</el-radio>
            <el-radio label="rejected">拒绝</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="审批意见">
          <el-input
            v-model="formData.comment"
            type="textarea"
            :rows="4"
            placeholder="请输入审批意见"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmitApproval">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { View, Position, Check, Close, Clock, User, Monitor, Money, CircleCheck, SuccessFilled, Finished, CloseBold } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'
import { getPendingApprovals, approveWorkflow, rejectWorkflow, createWorkflow, getStatistics } from '@/api/approval'
import { getPendingStatusChangeApprovals, approveStatusChangeRequest, rejectStatusChangeRequest, getContracts } from '@/api/contract'
import { getRoleText } from '@/utils/constants'

const router = useRouter()
const userStore = useUserStore()

const userRole = computed(() => getRoleText(userStore.userInfo?.role))
const isAdmin = computed(() => ['admin', 'contract_admin'].includes(userStore.userInfo?.role))

const loading = ref(false)
const statusChangeLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref(null)
const tableData = ref([])
const statusChangeData = ref([])
const currentContract = ref({})
const currentApprovalId = ref(null)
const currentWorkflowId = ref(null)

const stats = reactive({
  pendingCount: 0,
  salesPending: 0,
  techPending: 0,
  financePending: 0,
  rejectedCount: 0,
  approvedCount: 0,
  activeCount: 0,
  completedCount: 0
})

const formData = reactive({
  status: 'approved',
  comment: '',
  level: 1
})

const getStatusText = (status) => {
  const map = {
    draft: '草稿',
    pending: '待审批',
    approved: '已批准',
    active: '已生效',
    in_progress: '执行中',
    pending_pay: '待付款',
    completed: '已完成',
    terminated: '已终止',
    archived: '已归档'
  }
  return map[status] || status
}

const formatDateTime = (dateStr) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  if (isNaN(date.getTime())) return dateStr
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = date.getHours()
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const ampm = hours < 12 ? '上午' : '下午'
  const hour12 = hours % 12 || 12
  return `${year}-${month}-${day} ${ampm}${hour12}:${minutes}`
}

const getContractStatusType = (status) => {
  const typeMap = {
    draft: 'info',
    pending: 'warning',
    active: 'success',
    approved: 'success',
    rejected: 'danger'
  }
  return typeMap[status] || ''
}

const getContractStatusText = (status) => {
  const textMap = {
    draft: '草稿',
    pending: '待审批',
    active: '进行中',
    approved: '已批准',
    rejected: '已拒绝'
  }
  return textMap[status] || status
}

const getLevelType = (level) => {
  const typeMap = {
    1: 'warning',
    2: 'primary',
    3: 'success'
  }
  return typeMap[level] || 'info'
}

const getApprovalButtonText = (level) => {
  const textMap = {
    1: '销售总监审批',
    2: '技术总监审批',
    3: '财务总监审批'
  }
  return textMap[level] || '审批'
}

const getStatusTagType = (row) => {
  // 如果是pending，根据级别返回不同的颜色
  if (row.status === 'pending') {
    const levelTypeMap = {
      1: 'warning',    // 等待销售总监 - 橙色
      2: 'primary',   // 等待技术总监 - 蓝色
      3: 'success'    // 等待财务总监 - 绿色
    }
    return levelTypeMap[row.level] || 'info'
  }
  return getContractStatusType(row.status)
}

const getStatusDisplay = (row) => {
  if (row.status === 'pending') {
    const levelTextMap = {
      1: '待销售总监',
      2: '待技术总监',
      3: '待财务总监'
    }
    return levelTextMap[row.level] || '待审批'
  }
  return getContractStatusText(row.status)
}

const currentStepIndex = computed(() => {
  if (tableData.value.length === 0) return 0
  const firstItem = tableData.value[0]
  if (firstItem.workflow_status === 'completed' || firstItem.status === 'approved') {
    return 4 // 所有审批已完成
  }
  const level = firstItem.level || 1
  return level // level 1->1, 2->2, 3->3
})

const handleReject = (row) => {
  currentContract.value = row
  currentWorkflowId.value = row.workflow_id
  formData.level = row.level
  formData.status = 'rejected'
  dialogVisible.value = true
}

const loadData = async () => {
	loading.value = true
	try {
		const data = await getPendingApprovals()
		tableData.value = data || []
		calculateStats(data || [])
	} catch (error) {
		console.error('Failed to load approvals:', error)
		tableData.value = []
	} finally {
		loading.value = false
	}

	if (isAdmin.value) {
		loadContractStats()
	}
  
  if (isAdmin.value) {
    statusChangeLoading.value = true
    try {
      const data = await getPendingStatusChangeApprovals()
      statusChangeData.value = data || []
    } catch (error) {
      console.error('Failed to load status changes:', error)
      statusChangeData.value = []
    } finally {
      statusChangeLoading.value = false
    }
  }
}

const calculateStats = (data) => {
	stats.pendingCount = data.length
	stats.salesPending = data.filter(item => item.level === 1).length
	stats.techPending = data.filter(item => item.level === 2).length
	stats.financePending = data.filter(item => item.level === 3).length
}

const loadContractStats = async () => {
	try {
		const allRes = await getContracts({})
		if (Array.isArray(allRes)) {
			const contracts = allRes
			stats.rejectedCount = contracts.filter(c => c.status === 'terminated').length
			stats.activeCount = contracts.filter(c => c.status === 'active').length
			stats.completedCount = contracts.filter(c => c.status === 'completed')
			stats.approvedCount = contracts.filter(c => c.status === 'approved').length
		}
	} catch (error) {
		console.error('Failed to load contract stats:', error)
	}
}

const handleViewStatusChange = (row) => {
  router.push(`/contracts/${row.contract_id}`)
}

const handleApproveStatusChange = async (row) => {
  try {
    await ElMessageBox.confirm(`确定通过将合同 "${row.contract.title}" 状态变更为 "${getStatusText(row.to_status)}" 吗？`, '审批确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'success'
    })
    await approveStatusChangeRequest(row.id, { comment: '同意' })
    ElMessage.success('审批通过')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '操作失败')
    }
  }
}

const handleRejectStatusChange = async (row) => {
  try {
    await ElMessageBox.confirm(`确定拒绝将合同 "${row.contract.title}" 状态变更申请吗？`, '审批确认', {
      confirmButtonText: '确定拒绝',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await rejectStatusChangeRequest(row.id, { comment: '拒绝' })
    ElMessage.success('已拒绝')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '操作失败')
    }
  }
}

const handleView = (row) => {
  router.push(`/contracts/${row.contract_id || row.id}`)
}

const handleSubmit = async (row) => {
  try {
    await ElMessageBox.confirm(`确定提交合同 "${row.title}" 进行审批吗？`, '提交审批', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await createWorkflow({
      contract_id: row.id,
      creator_role: userStore.userInfo?.role
    })
    ElMessage.success('已提交审批')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '提交失败')
    }
  }
}

const handleApprove = async (row) => {
  currentContract.value = row
  currentApprovalId.value = row.approval_id
  currentWorkflowId.value = row.workflow_id
  formData.status = 'approved'
  formData.comment = ''
  formData.level = row.level
  dialogVisible.value = true
}

const handleSubmitApproval = async () => {
  const rejectedLevel = formData.status === 'rejected' ? formData.level : null
  try {
    if (formData.status === 'approved') {
      await approveWorkflow({
        workflow_id: currentWorkflowId.value,
        level: formData.level,
        comment: formData.comment
      })
      ElMessage.success('审批通过')
    } else {
      if (!formData.comment || !formData.comment.trim()) {
        ElMessage.error('请填写拒绝理由')
        return
      }
      await rejectWorkflow({
        workflow_id: currentWorkflowId.value,
        level: formData.level,
        comment: formData.comment
      })
      ElMessage.success('审批拒绝')
    }
    dialogVisible.value = false
    
    if (rejectedLevel) {
      const levelText = { 1: '销售总监', 2: '技术总监', 3: '财务总监' }[rejectedLevel]
      ElMessage.info(`合同已被${levelText}拒绝，需要重新从销售总监提交审批`)
      
      const currentPending = stats.pendingCount
      const currentLevelPending = rejectedLevel === 1 ? stats.salesPending : 
                                   rejectedLevel === 2 ? stats.techPending : 
                                   stats.financePending
      
      stats.pendingCount = currentPending
      stats.salesPending = rejectedLevel === 1 ? currentLevelPending - 1 : stats.salesPending + 1
      stats.techPending = rejectedLevel === 2 ? currentLevelPending - 1 : stats.techPending
      stats.financePending = rejectedLevel === 3 ? currentLevelPending - 1 : stats.financePending
    }
    
    loadData()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '操作失败')
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.approval-page {
  padding: 16px;
}

.approval-stats {
  margin-bottom: 16px;
}

.stat-card {
  border-radius: 8px;
}

.stat-card.rejected {
  border-left: 4px solid #f56c6c;
}

.stat-card.approved {
  border-left: 4px solid #67c23a;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.stat-icon.pending {
  background: #fef0f0;
  color: #e6a23c;
}

.stat-icon.sales {
  background: #f0f9ff;
  color: #409eff;
}

.stat-icon.tech {
  background: #f0f9eb;
  color: #67c23a;
}

.stat-icon.finance {
  background: #fdf6ec;
  color: #e6a23c;
}

.stat-icon.rejected {
  background: #fef0f0;
  color: #f56c6c;
}

.stat-icon.approved {
  background: #f0f9eb;
  color: #67c23a;
}

.stat-icon.active {
  background: #ecf5ff;
  color: #409eff;
}

.stat-icon.completed {
  background: #f0f9eb;
  color: #67c23a;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.2;
}

.stat-label {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}

.approval-card {
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  font-size: 18px;
  color: #409eff;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.flow-steps {
  padding: 12px 20px;
  background: linear-gradient(to right, #f0f9ff, #e6f4ff);
  border-radius: 8px;
  margin-bottom: 16px;
}

.approval-table {
  border-radius: 8px;
  overflow: hidden;
}

.contract-no {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #409eff;
  font-weight: 500;
}

.contract-title {
  color: #606266;
}

.amount {
  color: #e6a23c;
  font-weight: 600;
}

.level-info {
  display: flex;
  align-items: center;
}

.time {
  color: #909399;
  font-size: 13px;
}

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.action-buttons .el-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  transition: all 0.3s;
}

.approve-btn {
  color: #67c23a;
}

.approve-btn:hover {
  background: #f0f9ff;
}

.reject-btn {
  color: #f56c6c;
}

.reject-btn:hover {
  background: #fef0f0;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>