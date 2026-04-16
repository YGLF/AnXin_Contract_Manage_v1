package handlers

import (
	"contract-manage/crypto"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CryptoHandler struct {
	hsmService *crypto.HSMService
	sm4Service *crypto.SM4Service
	aesService *crypto.AESService
}

func NewCryptoHandler() *CryptoHandler {
	return &CryptoHandler{}
}

func (h *CryptoHandler) SetHSMService(endpoint, appID string) {
	h.hsmService = crypto.NewHSMService(endpoint, appID)
}

func (h *CryptoHandler) SetSM4Service(key string) error {
	service, err := crypto.NewSM4Service(key)
	if err != nil {
		return err
	}
	h.sm4Service = service
	return nil
}

func (h *CryptoHandler) SetAESService(key string) error {
	service, err := crypto.NewAESService(key)
	if err != nil {
		return err
	}
	h.aesService = service
	return nil
}

type EncryptRequest struct {
	Data     string `json:"data" binding:"required"`
	KeyID    string `json:"key_id"`
	AlgoType string `json:"algo_type"` // sm4, aes, hsm
}

type EncryptResponse struct {
	CipherText string `json:"cipher_text"`
	Algorithm  string `json:"algorithm"`
}

func (h *CryptoHandler) Encrypt(c *gin.Context) {
	var req EncryptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result string
	var err error
	algo := "sm4"

	switch req.AlgoType {
	case "hsm":
		if h.hsmService == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "HSM service not configured"})
			return
		}
		if req.KeyID != "" {
			result, err = h.hsmService.EncryptWithKey(req.Data, req.KeyID)
		} else {
			result, err = h.hsmService.Encrypt(req.Data)
		}
		algo = "hsm"
	case "aes":
		if h.aesService == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "AES service not configured"})
			return
		}
		result, err = h.aesService.EncryptCBC(req.Data)
		algo = "aes"
	default:
		if h.sm4Service == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SM4 service not configured"})
			return
		}
		result, err = h.sm4Service.Encrypt(req.Data)
		algo = "sm4"
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, EncryptResponse{
		CipherText: result,
		Algorithm:  algo,
	})
}

func (h *CryptoHandler) Decrypt(c *gin.Context) {
	var req EncryptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result string
	var err error

	switch req.AlgoType {
	case "hsm":
		if h.hsmService == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "HSM service not configured"})
			return
		}
		if req.KeyID != "" {
			result, err = h.hsmService.DecryptWithKey(req.Data, req.KeyID)
		} else {
			result, err = h.hsmService.Decrypt(req.Data)
		}
	case "aes":
		if h.aesService == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "AES service not configured"})
			return
		}
		result, err = h.aesService.DecryptCBC(req.Data)
	default:
		if h.sm4Service == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SM4 service not configured"})
			return
		}
		result, err = h.sm4Service.Decrypt(req.Data)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plain_text": result})
}

type KeyGenerateRequest struct {
	KeyType  string `json:"key_type"` // sm4, aes, hsm
	KeyID    string `json:"key_id"`
	KeyUsage string `json:"key_usage"` // encrypt, sign, mac
}

type KeyGenerateResponse struct {
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	KeyData   string `json:"key_data,omitempty"`
}

func (h *CryptoHandler) GenerateKey(c *gin.Context) {
	var req KeyGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.KeyType {
	case "hsm":
		if h.hsmService == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "HSM service not configured"})
			return
		}
		keyData, err := h.hsmService.GenerateKey(req.KeyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, KeyGenerateResponse{
			KeyID:     req.KeyID,
			Algorithm: "hsm",
			KeyData:   keyData,
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "key type not supported"})
	}
}

type HSMConfigRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
	AppID    string `json:"app_id" binding:"required"`
}

func (h *CryptoHandler) ConfigHSM(c *gin.Context) {
	var req HSMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.SetHSMService(req.Endpoint, req.AppID)
	c.JSON(http.StatusOK, gin.H{"message": "HSM configured successfully"})
}

type SM4ConfigRequest struct {
	Key string `json:"key" binding:"required"`
}

func (h *CryptoHandler) ConfigSM4(c *gin.Context) {
	var req SM4ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.SetSM4Service(req.Key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SM4 configured successfully"})
}

type AESConfigRequest struct {
	Key string `json:"key" binding:"required"`
}

func (h *CryptoHandler) ConfigAES(c *gin.Context) {
	var req AESConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.SetAESService(req.Key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "AES configured successfully"})
}

func (h *CryptoHandler) GetCryptoStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"hsm_configured": h.hsmService != nil,
		"sm4_configured": h.sm4Service != nil,
		"aes_configured": h.aesService != nil,
	})
}
