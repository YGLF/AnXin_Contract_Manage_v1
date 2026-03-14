<template>
  <router-view />
</template>

<script setup>
</script>

<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB',
    'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

body {
  background-color: #F8FAFC;
  color: #1E293B;
}

/* 全局滚动条美化 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: #F1F5F9;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb {
  background: #CBD5E1;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #94A3B8;
}

/* Element Plus 主题定制 */
:root {
  --el-color-primary: #6366F1;
  --el-color-primary-light-3: #818CF8;
  --el-color-primary-light-5: #A5B4FC;
  --el-color-primary-light-7: #C7D2FE;
  --el-color-primary-light-9: #E0E7FF;
  --el-color-primary-dark-2: #4F46E5;
  --el-color-success: #10B981;
  --el-color-warning: #F59E0B;
  --el-color-danger: #EF4444;
  --el-color-error: #EF4444;
  --el-border-radius-base: 10px;
  --el-border-radius-small: 8px;
  --el-font-size-base: 14px;
}

/* 按钮样式 */
.el-button {
  font-weight: 500;
  transition: all 0.2s ease;
}

.el-button--primary {
  background: linear-gradient(135deg, #6366F1 0%, #8B5CF6 100%);
  border: none;
}

.el-button--primary:hover {
  background: linear-gradient(135deg, #4F46E5 0%, #7C3AED 100%);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}

/* 卡片样式 */
.el-card {
  border: none;
  border-radius: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.06);
}

.el-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

/* 输入框样式 */
.el-input__wrapper {
  border-radius: 10px;
  box-shadow: 0 0 0 1px #E2E8F0 inset;
  transition: all 0.2s ease;
}

.el-input__wrapper:hover {
  box-shadow: 0 0 0 1px #CBD5E1 inset;
}

.el-input__wrapper.is-focus {
  box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2), 0 0 0 1px #6366F1 inset;
}

/* 表格样式 */
.el-table {
  --el-table-border-color: #F1F5F9;
  --el-table-header-bg-color: #F8FAFC;
}

.el-table th {
  font-weight: 600;
  color: #64748B;
}

.el-table__body tr:hover > td {
  background: #F8FAFC !important;
}

/* 标签样式 */
.el-tag {
  border-radius: 6px;
  font-weight: 500;
}

/* 下拉菜单 */
.el-dropdown-menu {
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
  border: none;
}

.el-dropdown-menu__item {
  border-radius: 8px;
  margin: 4px 8px;
  padding: 8px 16px;
}

/* 消息提示 */
.el-message {
  border-radius: 12px;
}

/* 对话框 */
.el-dialog {
  border-radius: 20px;
}

.el-dialog__header {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #F1F5F9;
}

.el-dialog__title {
  font-weight: 600;
  color: #1E293B;
}

/* 分页 */
.el-pagination {
  --el-pagination-button-bg-color: #F8FAFC;
}

.el-pagination.is-background .el-pager li:not(.is-disabled).is-active {
  background: linear-gradient(135deg, #6366F1 0%, #8B5CF6 100%);
}

/* 页面切换动画 */
.page-enter-active,
.page-leave-active {
  transition: all 0.3s ease;
}

.page-enter-from,
.page-leave-to {
  opacity: 0;
  transform: translateY(10px);
}

/* 选中文本颜色 */
::selection {
  background: rgba(99, 102, 241, 0.2);
  color: #1E293B;
}
</style>
