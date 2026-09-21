package auth

import (
	"errors"
	"net/http"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	srv "github.com/Secreto31126/tesis/common/services/auth"
	"github.com/gin-gonic/gin"
)

const REFRESH_COOKIE = "tiu_refresh"

// Cookies is how the refresh token travels: only to the auth routes, never
// to scripts, and over TLS unless a plain-HTTP dev setup says otherwise.
type Cookies struct {
	Secure bool
	Path   string
}

type Controller struct {
	service *srv.AuthService
	cookies Cookies
}

func NewController(service *srv.AuthService, cookies Cookies) *Controller {
	return &Controller{service, cookies}
}

func (ctrl *Controller) RegisterRoutes(router gin.IRouter) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", ctrl.Register)
		auth.POST("/verify-email", ctrl.VerifyEmail)
		auth.POST("/login", ctrl.Login)
		auth.POST("/refresh", middleware.SameOrigin(), ctrl.Refresh)
		auth.POST("/logout", middleware.SameOrigin(), ctrl.Logout)
		auth.POST("/password/forgot", ctrl.ForgotPassword)
		auth.POST("/password/reset", ctrl.ResetPassword)
	}
}

// Register godoc
// @Summary      Create an account with email and password
// @Description  Always answers 202: whether the address was already taken is only said by mail.
// @Tags         auth
// @Accept       json
// @Param        body  body  RegisterRequest  true  "Credentials and display name"
// @Success      202
// @Failure      400
// @Failure      429
// @Router       /auth/register [post]
func (ctrl *Controller) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	if err := ctrl.service.Register(req.Email, req.Password, req.Name, origin(c), c.ClientIP()); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{})
}

// VerifyEmail godoc
// @Summary      Confirm an address with the mailed token and start a session
// @Tags         auth
// @Accept       json
// @Param        body  body  TokenRequest  true  "Token from the verification mail"
// @Success      200  {object}  SessionResponse
// @Failure      400
// @Router       /auth/verify-email [post]
func (ctrl *Controller) VerifyEmail(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	tokens, err := ctrl.service.VerifyEmail(req.Token)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	ctrl.session(c, tokens)
}

// Login godoc
// @Summary      Sign in with email and password
// @Tags         auth
// @Accept       json
// @Param        body  body  LoginRequest  true  "Credentials"
// @Success      200  {object}  SessionResponse
// @Failure      401  "invalid_credentials"
// @Failure      403  "email_not_verified"
// @Failure      429
// @Router       /auth/login [post]
func (ctrl *Controller) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	tokens, err := ctrl.service.Login(req.Email, req.Password, c.ClientIP())
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	ctrl.session(c, tokens)
}

// Refresh godoc
// @Summary      Rotate the refresh cookie and mint a new access token
// @Tags         auth
// @Success      200  {object}  SessionResponse
// @Failure      401  "The cookie is missing, spent or revoked; it is cleared"
// @Router       /auth/refresh [post]
func (ctrl *Controller) Refresh(c *gin.Context) {
	refresh, err := c.Cookie(REFRESH_COOKIE)
	if err != nil {
		ctrl.clearCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tokens, err := ctrl.service.Refresh(refresh)
	if err != nil {
		ctrl.clearCookie(c)
		ctrl.fail(c, err)
		return
	}

	ctrl.session(c, tokens)
}

// Logout godoc
// @Summary      Revoke the session behind the refresh cookie
// @Tags         auth
// @Success      204
// @Router       /auth/logout [post]
func (ctrl *Controller) Logout(c *gin.Context) {
	if refresh, err := c.Cookie(REFRESH_COOKIE); err == nil {
		if err := ctrl.service.Logout(refresh); err != nil {
			ctrl.fail(c, err)
			return
		}
	}

	ctrl.clearCookie(c)
	c.Status(http.StatusNoContent)
}

// ForgotPassword godoc
// @Summary      Mail a password reset link
// @Description  Always answers 202, known address or not.
// @Tags         auth
// @Accept       json
// @Param        body  body  EmailRequest  true  "Address"
// @Success      202
// @Failure      429
// @Router       /auth/password/forgot [post]
func (ctrl *Controller) ForgotPassword(c *gin.Context) {
	var req EmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	if err := ctrl.service.ForgotPassword(req.Email, origin(c), c.ClientIP()); err != nil {
		ctrl.fail(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{})
}

// ResetPassword godoc
// @Summary      Set a new password with the mailed token
// @Description  Verifies the address, logs every other session out and starts a new one.
// @Tags         auth
// @Accept       json
// @Param        body  body  ResetRequest  true  "Token and new password"
// @Success      200  {object}  SessionResponse
// @Failure      400
// @Router       /auth/password/reset [post]
func (ctrl *Controller) ResetPassword(c *gin.Context) {
	var req ResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	tokens, err := ctrl.service.ResetPassword(req.Token, req.Password)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	ctrl.session(c, tokens)
}

func (ctrl *Controller) session(c *gin.Context, tokens *srv.Tokens) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(REFRESH_COOKIE, tokens.Refresh, int(tokens.RefreshTTL.Seconds()), ctrl.cookies.Path, "", ctrl.cookies.Secure, true)

	c.JSON(http.StatusOK, SessionResponse{
		AccessToken: tokens.Access,
		TokenType:   "Bearer",
		ExpiresIn:   int(tokens.ExpiresIn.Seconds()),
		User:        tokens.User.Profile(),
	})
}

func (ctrl *Controller) clearCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(REFRESH_COOKIE, "", -1, ctrl.cookies.Path, "", ctrl.cookies.Secure, true)
}

func (ctrl *Controller) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, srv.ErrInvalidEmail), errors.Is(err, srv.ErrWeakPassword), errors.Is(err, srv.ErrInvalidName):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, srv.ErrInvalidToken):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_token"})
	case errors.Is(err, srv.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
	case errors.Is(err, srv.ErrInvalidSession):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case errors.Is(err, srv.ErrEmailNotVerified):
		c.JSON(http.StatusForbidden, gin.H{"error": "email_not_verified"})
	case errors.Is(err, srv.ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// origin is where the browser is, which is where mailed links must point:
// the proxy tells us the scheme, the Host header the rest.
func origin(c *gin.Context) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}
	return scheme + "://" + c.Request.Host
}
