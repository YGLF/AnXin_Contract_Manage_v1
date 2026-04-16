<template>
  <div class="contract-page">
    <!-- 搜索区域 -->
    <div class="search-bar">
      <el-input v-model="searchForm.title" placeholder="合同标题" clearable style="width: 160px" size="default" />
      <el-select v-model="searchForm.status" placeholder="状态" clearable style="width: 120px" size="default">
        <el-option label="草稿" value="draft" />
        <el-option label="待审批" value="pending" />
        <el-option label="已批准" value="approved" />
        <el-option label="进行中" value="active" />
        <el-option label="已完成" value="completed" />
      </el-select>
      <el-button type="primary" size="default" @click="handleSearch">查询</el-button>
      <el-button size="default" @click="handleReset">重置</el-button>
      <div class="search-right">
        <el-button type="primary" size="default" @click="handleAdd">
          <el-icon><Plus /></el-icon> 新增合同
        </el-button>
      </div>
    </div>

    <el-card class="table-card">
      <el-table :data="tableData" style="width: 100%" v-loading="loading" :cell-style="{ padding: '8px 0' }" :row-class-name="getRowClassName">
        <el-table-column prop="contract_no" label="合同编号" min-width="110" />
        <el-table-column prop="title" label="合同标题" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="title-cell">
              <span>{{ row.title }}</span>
              <el-tag v-if="row.reminders && row.reminders.length > 0" size="small" type="warning" class="reminder-tag">已提醒</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="customer" label="客户" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.customer?.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="creator" label="销售人员" min-width="90">
          <template #default="{ row }">
            <span v-if="row.creator">{{ row.creator.full_name || row.creator.username }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="amount" label="金额" min-width="100">
          <template #default="{ row }">
            ¥{{ row.amount?.toLocaleString() }}
          </template>
        </el-table-column>
<el-table-column prop="status" label="状态" min-width="80">
    <template #default="{ row }">
      <el-tag :type="getStatusType(row.status)" size="small">{{ getStatusText(row.status) }}</el-tag>
    </template>
  </el-table-column>
  <el-table-column prop="workflow" label="审批进度" min-width="140">
    <template #default="{ row }">
      <div v-if="row.workflow_status && row.workflow_status.has_workflow" class="workflow-cell">
        <el-progress
          :percentage="Math.round((row.workflow_status.current_level / row.workflow_status.max_level) * 100)"
          :status="getWorkflowProgressStatus(row.workflow_status.status)"
          :stroke-width="4"
          style="width: 80px"
        />
        <span class="workflow-text">第{{ row.workflow_status.current_level }}/{{ row.workflow_status.max_level }}级</span>
        <el-button
          v-if="row.workflow_status.status === 'pending'"
          type="warning"
          link
          size="small"
          @click="handleSendReminder(row)"
        >
          <el-icon><Bell /></el-icon>
        </el-button>
      </div>
      <span v-else class="text-gray">-</span>
    </template>
  </el-table-column>
  <el-table-column prop="sign_date" label="签约日期" min-width="100">
    <template #default="{ row }">
      {{ formatDate(row.sign_date) }}
    </template>
  </el-table-column>
  <el-table-column prop="end_date" label="到期日期" min-width="100">
    <template #default="{ row }">
      {{ formatDate(row.end_date) }}
    </template>
  </el-table-column>
<el-table-column label="操作" width="120" fixed="right">
  <template #default="{ row }">
    <div class="action-buttons">
      <el-tooltip content="查看详情" placement="top">
        <el-button type="primary" link @click="handleView(row)">
          <el-icon><View /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="编辑" placement="top">
        <el-button type="warning" link @click="handleEdit(row)">
          <el-icon><Edit /></el-icon>
        </el-button>
      </el-tooltip>
      <el-tooltip content="删除" placement="top">
        <el-button type="danger" link @click="handleDelete(row)">
          <el-icon><Delete /></el-icon>
        </el-button>
      </el-tooltip>
    </div>
  </template>
</el-table-column>
</el-table>

<el-pagination
  v-model:current-page="pagination.page"
  v-model:page-size="pagination.size"
  :page-sizes="[10, 20, 50, 100]"
  :total="pagination.total"
  layout="total, sizes, prev, pager, next, jumper"
  @size-change="loadData"
  @current-change="loadData"
  style="margin-top: 12px; justify-content: flex-end"
/>
</el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="800px"
      @close="handleDialogClose"
    >
      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="120px">
        <el-form-item label="合同标题" prop="title">
          <el-input v-model="formData.title" placeholder="请输入合同标题" />
        </el-form-item>
        <el-form-item label="客户" prop="customer_id">
          <el-select v-model="formData.customer_id" placeholder="请选择客户" style="width: 100%">
            <el-option
              v-for="customer in customers"
              :key="customer.id"
              :label="customer.name"
              :value="customer.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="合同类型" prop="contract_type_id">
          <el-select v-model="formData.contract_type_id" placeholder="请选择合同类型" style="width: 100%">
            <el-option
              v-for="type in contractTypes"
              :key="type.id"
              :label="type.name"
              :value="type.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="金额" prop="amount">
          <el-input-number v-model="formData.amount" :precision="2" :min="0" style="width: 100%" />
        </el-form-item>
        <el-form-item label="签约日期" prop="sign_date">
          <el-date-picker
            v-model="formData.sign_date"
            type="date"
            placeholder="请选择签约日期"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="开始日期" prop="start_date">
          <el-date-picker
            v-model="formData.start_date"
            type="date"
            placeholder="请选择开始日期"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="结束日期" prop="end_date">
          <el-date-picker
            v-model="formData.end_date"
            type="date"
            placeholder="请选择结束日期"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="付款条件" prop="payment_terms">
          <el-input
            v-model="formData.payment_terms"
            type="textarea"
            :rows="3"
            placeholder="请输入付款条件"
          />
        </el-form-item>
        <el-form-item label="合同内容" prop="content">
          <el-input
            v-model="formData.content"
            type="textarea"
            :rows="5"
            placeholder="请输入合同内容"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, View, Edit, Delete, Bell } from '@element-plus/icons-vue'
import { getContractList, getContractDetail, createContract, updateContract, deleteContract } from '@/api/contract'
import { getCustomerList } from '@/api/customer'
import { getContractTypeList } from '@/api/customer'
import { getWorkflowStatus, sendApprovalReminder } from '@/api/approval'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const formRef = ref(null)
const tableData = ref([])
const customers = ref([])
const contractTypes = ref([])

const searchForm = reactive({
  title: '',
  status: ''
})

const pagination = reactive({
  page: 1,
  size: 10,
  total: 0
})

const formData = reactive({
  title: '',
  customer_id: null,
  contract_type_id: null,
  amount: null,
  sign_date: '',
  start_date: '',
  end_date: '',
  payment_terms: '',
  content: ''
})

const formRules = {
  title: [{ required: true, message: '请输入合同标题', trigger: 'blur' }],
  customer_id: [{ required: true, message: '请选择客户', trigger: 'change' }],
  contract_type_id: [{ required: true, message: '请选择合同类型', trigger: 'change' }]
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  if (isNaN(date.getTime())) return dateStr
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const getStatusType = (status) => {
  const typeMap = {
    draft: 'info',
    pending: 'warning',
    approved: 'success',
    active: 'primary',
    completed: 'success',
    terminated: 'danger'
  }
  return typeMap[status] || ''
}

const getStatusText = (status) => {
  const textMap = {
    draft: '草稿',
    pending: '待审批',
    approved: '已批准',
    active: '进行中',
    completed: '已完成',
    terminated: '已终止'
  }
  return textMap[status] || status
}

const getWorkflowProgressStatus = (status) => {
  if (status === 'approved' || status === 'completed') return 'success'
  if (status === 'rejected') return 'exception'
  return 'warning'
}

const getLevelName = (level) => {
  const nameMap = {
    1: '销售总监',
    2: '技术总监',
    3: '财务总监'
  }
  return nameMap[level] || '未知'
}

const getRoleName = (role) => {
  const nameMap = {
    'sales_director': '销售',
    'tech_director': '技术',
    'finance_director': '财务'
  }
  return nameMap[role] || role
}

const getStepClass = (node) => {
  if (node.status === 'approved') return 'step-approved'
  if (node.status === 'rejected') return 'step-rejected'
  if (node.status === 'pending') return 'step-pending'
  return ''
}

const getRowClassName = ({ row }) => {
  if (row.reminders && row.reminders.length > 0) {
    return 'reminder-row'
  }
  return ''
}

const loadWorkflowStatus = async (contractId) => {
  try {
    const status = await getWorkflowStatus(contractId)
    return status
  } catch (error) {
    return null
  }
}

const loadData = async () => {
  loading.value = true
  try {
    const params = {
      skip: (pagination.page - 1) * pagination.size,
      limit: pagination.size
    }
    if (searchForm.title) {
      params.title = searchForm.title
    }
    if (searchForm.status) {
      params.status = searchForm.status
    }
    const res = await getContractList(params)
    
    const data = res.data || res
    const total = res.total || data.length
    
    // 加载每个合同的工作流状态
    for (const contract of data) {
      if (contract.status === 'pending') {
        contract.workflow_status = await loadWorkflowStatus(contract.id)
      }
    }
    
    tableData.value = data
    pagination.total = total
  } finally {
    loading.value = false
  }
}

const handleSendReminder = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要催办合同 "${row.title}" 的审批吗？\n提醒将发送给当前待审批的负责人。`,
      '催办审批',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await sendApprovalReminder(row.id)
    ElMessage.success('催办提醒已发送')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '发送提醒失败')
    }
  }
}

const loadCustomers = async () => {
  try {
    const res = await getCustomerList({ limit: 1000 })
    customers.value = res.data || res || []
  } catch (e) {
    customers.value = []
  }
}

const loadContractTypes = async () => {
  try {
    const res = await getContractTypeList({ limit: 1000 })
    contractTypes.value = res.data || res || []
  } catch (e) {
    contractTypes.value = []
  }
}

const handleAdd = () => {
  dialogTitle.value = '新增合同'
  dialogVisible.value = true
}

const handleEdit = (row) => {
  dialogTitle.value = '编辑合同'
  Object.assign(formData, row)
  dialogVisible.value = true
}

const handleView = (row) => {
  router.push(`/contracts/${row.id}`)
}

const handleDelete = async (row) => {
  await ElMessageBox.confirm('确定要删除该合同吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  await deleteContract(row.id)
  ElMessage.success('删除成功')
  loadData()
}

const handleSearch = () => {
  pagination.page = 1
  loadData()
}

const handleReset = () => {
  Object.assign(searchForm, { title: '', status: '' })
  handleSearch()
}

const handleSubmit = async () => {
  await formRef.value.validate(async (valid) => {
    if (valid) {
      if (formData.id) {
        await updateContract(formData.id, formData)
        ElMessage.success('更新成功')
      } else {
        await createContract(formData)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      loadData()
    }
  })
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
  Object.assign(formData, {
    title: '',
    customer_id: null,
    contract_type_id: null,
    amount: null,
    sign_date: '',
    start_date: '',
    end_date: '',
    payment_terms: '',
    content: ''
  })
}

onMounted(async () => {
  if (route.query.status) {
    searchForm.status = route.query.status
  }
  if (route.query.title) {
    searchForm.title = route.query.title
  }
  loadData()
  loadCustomers()
  loadContractTypes()
  
  if (route.query.action === 'create') {
    handleAdd()
    window.history.replaceState({}, '', '/contracts')
  } else if (route.query.action === 'edit' && route.query.id) {
    const id = parseInt(route.query.id)
    const data = await getContractDetail(id)
    handleEdit(data)
    window.history.replaceState({}, '', '/contracts')
  }
})
</script>

<style scoped>
.contract-page {
  padding: 16px;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.search-right {
  margin-left: auto;
}

.table-card {
  border-radius: 8px;
}

.workflow-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.workflow-text {
  font-size: 12px;
  color: #606266;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
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
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.workflow-info {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.workflow-level-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.level-badge {
  background: #409eff;
  color: white;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: bold;
}

.level-text {
  font-size: 13px;
  color: #303133;
  font-weight: 500;
}

.workflow-steps {
  display: flex;
  gap: 4px;
  margin-top: 6px;
  font-size: 11px;
}

.step-item {
  padding: 2px 6px;
  border-radius: 4px;
  background: #e4e7ed;
  color: #606266;
}

.step-approved {
  background: #67c23a;
  color: white;
}

.step-rejected {
  background: #f56c6c;
  color: white;
}

.step-pending {
  background: #e6a23c;
  color: white;
}

.text-gray {
  color: #909399;
}

.title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.reminder-tag {
  flex-shrink: 0;
}

:deep(.reminder-row) {
  background-color: #fdf6ec !important;
}

:deep(.reminder-row:hover) {
  background-color: #faecd8 !important;
}
</style>