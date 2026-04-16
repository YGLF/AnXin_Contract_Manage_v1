package handlers

import (
	"archive/zip"
	"contract-manage/config"
	"contract-manage/middleware"
	"contract-manage/models"
	"contract-manage/services"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// generateSafeFilename 生成安全的随机文件名
// 防止路径遍历攻击
func generateSafeFilename(originalName string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return hex.EncodeToString(randBytes) + ext
}

// ContractHandler 合同处理器
// 处理合同的CRUD操作、状态变更、文档管理等请求
type ContractHandler struct {
	contractService *services.ContractService // 合同服务实式
	db              *gorm.DB                  // 数据库实例
}

// NewContractHandler 创建合同处理器实例
// 返回：配置好的ContractHandler指针
func NewContractHandler(db *gorm.DB) *ContractHandler {
	return &ContractHandler{
		contractService: services.NewContractService(),
		db:              db,
	}
}

// GetContracts 获取合同列表处理器
// 支持分页、多条件筛选和角色可见性控制
// GET /api/contracts
func (h *ContractHandler) GetContracts(c *gin.Context) {
	skip, _ := strconv.Atoi(c.DefaultQuery("skip", "0"))                        // 跳过记录数，默认0
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))                    // 每页数量，默认100
	customerID, _ := strconv.ParseUint(c.Query("customer_id"), 10, 32)          // 按客户筛选
	contractTypeID, _ := strconv.ParseUint(c.Query("contract_type_id"), 10, 32) // 按合同类型筛选
	status := c.Query("status")                                                 // 按状态筛选
	title := c.Query("title")                                                   // 按标题搜索

	// 获取当前用户信息用于可见性过滤
	userID, _ := middleware.GetCurrentUserID(c)
	role, _ := middleware.GetCurrentUserRole(c)

	visibility := &services.ContractVisibilityParams{
		UserID: userID,
		Role:   role,
	}

	contracts, err := h.contractService.GetContracts(skip, limit, uint(customerID), uint(contractTypeID), status, title, visibility)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取总数
	count, _ := h.contractService.GetContractsCount(uint(customerID), uint(contractTypeID), status, title, visibility)

	c.JSON(http.StatusOK, gin.H{
		"data":  contracts,
		"total": count,
	})
}

// GetContractByID 获取合同详情处理器
// 返回指定合同的完整信息
// GET /api/contracts/:contract_id
func (h *ContractHandler) GetContractByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	contract, err := h.contractService.GetContractByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
		return
	}

	c.JSON(http.StatusOK, contract)
}

// CreateContract 创建合同处理器
// 添加新的合同记录
// POST /api/contracts
func (h *ContractHandler) CreateContract(c *gin.Context) {
	var input services.ContractCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	contract, err := h.contractService.CreateContract(input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, contract)
}

// UpdateContract 更新合同处理器
// 修改合同的基本信息
// PUT /api/contracts/:contract_id
func (h *ContractHandler) UpdateContract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	var input services.ContractUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contract, err := h.contractService.UpdateContract(uint(id), input)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
		return
	}

	c.JSON(http.StatusOK, contract)
}

// DeleteContract 删除合同处理器
// 删除指定的合同记录
// DELETE /api/contracts/:contract_id
func (h *ContractHandler) DeleteContract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	if err := h.contractService.DeleteContract(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetContractExecutions 获取合同执行记录列表处理器
// 返回合同关联的所有执行阶段和进度记录
// GET /api/contracts/:contract_id/executions
func (h *ContractHandler) GetContractExecutions(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	executions, err := h.contractService.GetContractExecutions(uint(contractID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// CreateContractExecution 创建合同执行记录处理器
// 添加合同的执行阶段或进度信息
// POST /api/contracts/:contract_id/executions
func (h *ContractHandler) CreateContractExecution(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	var input services.ContractExecutionCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.ContractID = uint(contractID)

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	execution, err := h.contractService.CreateContractExecution(input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, execution)
}

// DeleteExecution 删除执行记录处理器
// 删除指定的执行记录
// DELETE /api/executions/:execution_id
func (h *ContractHandler) DeleteExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("execution_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.contractService.DeleteExecution(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetContractDocuments 获取合同文档列表处理器
// 返回合同关联的所有附件文档
// GET /api/contracts/:contract_id/documents
func (h *ContractHandler) GetContractDocuments(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	documents, err := h.contractService.GetDocuments(uint(contractID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, documents)
}

// CreateContractDocument 上传合同文档处理器
// 为合同上传附件文档
// POST /api/contracts/:contract_id/documents
func (h *ContractHandler) CreateContractDocument(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}

	// 安全检查：验证文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".jpg", ".jpeg", ".png", ".zip"}
	isAllowed := false
	for _, e := range allowedExts {
		if ext == e {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型"})
		return
	}

	// 安全：生成随机文件名防止路径遍历
	filename := generateSafeFilename(file.Filename)
	uploadDir := config.AppConfig.UploadDir
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	filePath := fmt.Sprintf("%s/%d/%s", uploadDir, contractID, filename)

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	input := services.DocumentCreateInput{
		ContractID: uint(contractID),
		Name:       filename,
		FilePath:   "/" + filePath,
		FileSize:   int(file.Size),
		FileType:   filepath.Ext(filename)[1:],
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	document, err := h.contractService.CreateDocument(input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, document)
}

// PreviewDocument 预览文档处理器
// 根据不同文件类型返回预览内容
// 支持：PDF、图片、Word(.docx)提取文本、文本文件等
// GET /api/documents/:document_id/preview
func (h *ContractHandler) PreviewDocument(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("document_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	document, err := h.contractService.GetDocumentByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// 构建绝对文件路径
	absFilePath := filepath.Join(".", document.FilePath)

	// 检查文件是否存在
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found: " + absFilePath})
		return
	}

	// 根据文件类型返回不同的内容
	fileExt := strings.ToLower(filepath.Ext(document.Name))

	// Word 文档 (.docx) 返回纯文本内容
	if fileExt == ".docx" {
		text, err := extractTextFromDocx(absFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文档内容: " + err.Error()})
			return
		}

		// 返回纯文本内容
		c.JSON(http.StatusOK, gin.H{
			"document_id": document.ID,
			"file_name":   document.Name,
			"file_type":   document.FileType,
			"file_size":   document.FileSize,
			"created_at":  document.CreatedAt,
			"content":     text,
		})
		return
	}

	switch fileExt {
	case ".pdf":
		// PDF 文件直接返回
		c.Header("Content-Type", "application/pdf")
		c.File(absFilePath)
	case ".doc":
		// Word 文档返回文件内容
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", document.Name))
		c.File(absFilePath)
	case ".xls", ".xlsx":
		// Excel 文件返回文件内容
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", document.Name))
		c.File(absFilePath)
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		// 图片文件直接返回
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", document.Name))
		c.File(absFilePath)
	case ".txt":
		// 文本文件返回内容
		content, err := os.ReadFile(absFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件内容"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"document_id": document.ID,
			"file_name":   document.Name,
			"file_type":   document.FileType,
			"file_size":   document.FileSize,
			"created_at":  document.CreatedAt,
			"content":     string(content),
		})
		return
	case ".html", ".htm":
		// HTML 文件返回内容
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.File(absFilePath)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型: " + fileExt})
	}
}

// convertWordToHTML 将 Word 文档转换为 HTML
func (h *ContractHandler) convertWordToHTML(docxPath, filePath string) (string, error) {
	// 使用 mammoth 库转换 Word 到 HTML
	// 这里需要调用 Python 脚本
	// 由于 Go 调用 Python 比较复杂，我们可以使用 exec 执行 mammoth 命令行工具
	// 或者使用 Go 库

	// 简单实现：返回提示信息，实际部署时需要安装 mammoth 并调用
	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>文档预览</title>
			<style>
				body { font-family: Arial, sans-serif; margin: 20px; }
				.info { background: #f0f0f0; padding: 20px; border-radius: 5px; }
			</style>
		</head>
		<body>
			<div class="info">
				<h3>Word 文档预览</h3>
				<p>文件: %s</p>
				<p>Word 文档需要下载后查看完整内容。</p>
				<p><a href="%s" download>点击下载文件</a></p>
			</div>
		</body>
		</html>
	`, filepath.Base(docxPath), filePath), nil
}

// DeleteDocument 删除文档处理器
// 删除合同关联的附件文档及其物理文件
// DELETE /api/documents/:document_id
func (h *ContractHandler) DeleteDocument(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("document_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	if err := h.contractService.DeleteDocument(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetContractLifecycle 获取合同生命周期处理器
// 返回合同的所有状态变更历史事件
// GET /api/contracts/:contract_id/lifecycle
func (h *ContractHandler) GetContractLifecycle(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	events, err := h.contractService.GetLifecycleEvents(uint(contractID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// UpdateContractStatus 更新合同状态处理器
// 直接更新合同状态（用于不需要审批的状态变更）
// PUT /api/contracts/:contract_id/status
func (h *ContractHandler) UpdateContractStatus(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	var input struct {
		Status      string `json:"status" binding:"required"` // 目标状态
		Description string `json:"description"`               // 状态变更描述
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	contract, err := h.contractService.UpdateContractStatus(uint(contractID), input.Status, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contract)
}

// ArchiveContract 归档合同处理器
// 将合同标记为归档状态，归档需要管理员权限
// POST /api/contracts/:contract_id/archive
func (h *ContractHandler) ArchiveContract(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	contract, err := h.contractService.ArchiveContract(uint(contractID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contract)
}

// UploadContractTemplate 上传合同模板处理器
// 上传Word文档(.docx)格式的合同模板并解析内容
// POST /api/contracts/upload-template
func (h *ContractHandler) UploadContractTemplate(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".docx" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .docx 格式文件"})
		return
	}

	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小不能超过 10MB"})
		return
	}

	uploadDir := "./uploads/contracts"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename)
	filePath := filepath.Join(uploadDir, filename)

	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	parsedData, parseErr := parseDocxFile(filePath)

	if parseErr != nil {
		c.JSON(200, gin.H{
			"success":  true,
			"message":  "文件上传成功，但解析失败: " + parseErr.Error(),
			"file_url": "/uploads/contracts/" + filename,
			"data":     nil,
		})
		return
	}

	c.JSON(200, gin.H{
		"success":  true,
		"message":  "文件上传并解析成功",
		"file_url": "/uploads/contracts/" + filename,
		"data":     parsedData,
	})
}

// parseDocxFile 解析Word文档内容
// 从.docx文件中提取文本内容并解析合同信息
// 参数：filePath-文件路径
// 返回：解析后的合同数据映射
func parseDocxFile(filePath string) (map[string]interface{}, error) {
	text, err := extractTextFromDocx(filePath)
	if err != nil {
		return nil, err
	}

	if text == "" {
		return nil, fmt.Errorf("无法读取文档内容")
	}

	data := extractContractData(text)

	return data, nil
}

// extractTextFromDocx 从Word文档提取纯文本内容
// 通过解析docx的XML结构提取文本
// 参数：filePath-文件路径
// 返回：提取的纯文本内容
func extractTextFromDocx(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var text strings.Builder

	for _, file := range r.File {
		// docx文件的正文内容在word/document.xml中
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}

			// 正则匹配XML中的文本节点 <w:t>...</w:t>
			re := regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)
			matches := re.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					text.WriteString(match[1])
					text.WriteString(" ")
				}
			}
			break
		}
	}

	return text.String(), nil
}

// contentToString 将任意类型转换为字符串
func contentToString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// extractContractData 从合同文本中提取结构化数据
// 使用正则表达式匹配合同关键字段
// 参数：text-合同纯文本内容
// 返回：包含合同信息的键值对映射
func extractContractData(text string) map[string]interface{} {
	data := make(map[string]interface{})

	// 定义合同字段的正则匹配模式
	patterns := map[string]string{
		"contract_no":   `合同编号[：:]\s*([A-Z0-9\-]+)(?:\s|$|\n)`,
		"title":         `合同名称[：:]\s*([^\n]+?)\s*(?:\n|$)`,
		"customer_name": `甲方[（(]客户[）)][：:]\s*([^\n]+?)\s*(?:\n|$)`,
		"amount":        `合同金额[：:]\s*([\d,]+\.?\d*)\s*(?:元|万)?(?:\s|$|\n)`,
		"sign_date":     `签订日期[：:]\s*(\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日]?)(?:\s|$|\n)`,
		"start_date":    `开始日期[：:]\s*(\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日]?)(?:\s|$|\n)`,
		"end_date":      `结束日期[：:]\s*(\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日]?)(?:\s|$|\n)`,
		"contract_type": `合同类型[：:]\s*([^\n]+?)\s*(?:\n|$)`,
	}

	// 遍历所有模式进行匹配
	for key, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			value = strings.ReplaceAll(value, "年", "-")
			value = strings.ReplaceAll(value, "月", "-")
			value = strings.ReplaceAll(value, "日", "")

			switch key {
			case "amount":
				value = strings.ReplaceAll(value, ",", "")
				if num, err := strconv.ParseFloat(value, 64); err == nil {
					data[key] = num
				}
			case "sign_date", "start_date", "end_date":
				if isValidDate(value) {
					data[key] = formatDate(value)
				}
			default:
				if value != "" {
					data[key] = value
				}
			}
		}
	}

	// 从行中提取联系人信息
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "联系人") && strings.Contains(line, "：") {
			if match := regexp.MustCompile(`联系人[：:]\s*(.{2,20})`).FindStringSubmatch(line); len(match) > 1 {
				data["contact_person"] = strings.TrimSpace(match[1])
			}
		}
		if strings.Contains(line, "电话") && strings.Contains(line, "：") {
			if match := regexp.MustCompile(`电话[：:]\s*([\d\-]+)`).FindStringSubmatch(line); len(match) > 1 {
				data["contact_phone"] = strings.TrimSpace(match[1])
			}
		}
	}

	_ = models.DB

	return data
}

// isValidDate 验证日期格式是否有效
// 参数：date-日期字符串
// 返回：是否有效
func isValidDate(date string) bool {
	re := regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}$`)
	return re.MatchString(date)
}

// formatDate 格式化日期字符串
// 将各种格式的日期统一转换为YYYY-MM-DD格式
func formatDate(date string) string {
	date = strings.ReplaceAll(date, "/", "-")
	parts := strings.Split(date, "-")
	if len(parts) == 3 {
		return fmt.Sprintf("%s-%02s-%02s", parts[0], parts[1], parts[2])
	}
	return date
}

// CreateStatusChangeRequest 创建状态变更申请处理器
// 创建状态变更申请，如果目标状态不需要审批则直接变更
// POST /api/contracts/:contract_id/status-change
func (h *ContractHandler) CreateStatusChangeRequest(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	var input services.StatusChangeRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 检查目标状态是否需要审批
	if !h.contractService.IsStatusChangeRequireApproval(input.ToStatus) {
		// 不需要审批，直接更新状态
		contract, err := h.contractService.UpdateContractStatus(uint(contractID), input.ToStatus, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"direct": true, "contract": contract})
		return
	}

	// 需要审批，创建申请
	request, err := h.contractService.CreateStatusChangeRequest(uint(contractID), input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request == nil {
		// 状态已直接更新
		contract, err := h.contractService.UpdateContractStatus(uint(contractID), input.ToStatus, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"direct": true, "contract": contract})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"direct": false, "request": request})
}

// GetStatusChangeRequests 获取状态变更申请记录处理器
// 返回合同的所有状态变更申请历史
// GET /api/contracts/:contract_id/status-change
func (h *ContractHandler) GetStatusChangeRequests(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	requests, err := h.contractService.GetStatusChangeRequests(uint(contractID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

// GetPendingStatusChangeApprovals 获取待审批状态变更列表处理器
// 返回当前用户角色需要审批的状态变更申请
// GET /api/pending-status-changes
func (h *ContractHandler) GetPendingStatusChangeApprovals(c *gin.Context) {
	role, _ := middleware.GetCurrentUserRole(c)
	if role == "" {
		role = "user"
	}

	requests, err := h.contractService.GetPendingStatusChangeRequests(role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

// ApproveStatusChangeRequest 审批通过状态变更处理器
// 管理员/审计管理员审批状态变更申请
// POST /api/status-change-requests/:request_id/approve
func (h *ContractHandler) ApproveStatusChangeRequest(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("request_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var input struct {
		Comment string `json:"comment"` // 审批意见
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	result, err := h.contractService.ApproveStatusChangeRequest(uint(requestID), userID, input.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RejectStatusChangeRequest 拒绝状态变更处理器
// 管理员/审计管理员拒绝状态变更申请
// POST /api/status-change-requests/:request_id/reject
func (h *ContractHandler) RejectStatusChangeRequest(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("request_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var input struct {
		Comment string `json:"comment"` // 拒绝原因
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	result, err := h.contractService.RejectStatusChangeRequest(uint(requestID), userID, input.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
