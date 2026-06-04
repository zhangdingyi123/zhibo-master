package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/service"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type productBody struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	CoverURL    string   `json:"coverUrl"`
	Images      []string `json:"images"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var body productBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidProductName)
		return
	}
	user := middleware.CurrentUser(c)
	p, err := h.svc.Create(c.Request.Context(), user.ID, service.CreateProductInput{
		Name:        body.Name,
		Description: body.Description,
		CoverURL:    body.CoverURL,
		Images:      body.Images,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, p)
}

func (h *ProductHandler) List(c *gin.Context) {
	user := middleware.CurrentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var status *domain.ProductStatus
	if s := c.Query("status"); s != "" {
		st := domain.ProductStatus(s)
		status = &st
	}

	result, err := h.svc.List(c.Request.Context(), user.ID, service.ListProductsInput{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *ProductHandler) Get(c *gin.Context) {
	user := middleware.CurrentUser(c)
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	p, err := h.svc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *ProductHandler) Update(c *gin.Context) {
	var body productBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, domain.ErrInvalidProductName)
		return
	}
	user := middleware.CurrentUser(c)
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	p, err := h.svc.Update(c.Request.Context(), user.ID, id, service.UpdateProductInput{
		Name:        body.Name,
		Description: body.Description,
		CoverURL:    body.CoverURL,
		Images:      body.Images,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, p)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	user := middleware.CurrentUser(c)
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Fail(c, domain.ErrNotFound)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), user.ID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func parseID(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
