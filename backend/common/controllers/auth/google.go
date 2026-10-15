package auth

import (
	"net/http"

	srv "github.com/Secreto31126/tesis/common/services/auth"
	"github.com/gin-gonic/gin"
)

// GoogleStart godoc
// @Summary      Begin a Google sign-in
// @Description  Answers the consent URL to open and binds the attempt to this browser with the tiu_signin cookie. Google sends the user back to /auth/google/callback on this origin.
// @Tags         auth
// @Produce      json
// @Param        next  query  string  false  "Path to land on afterwards, relative to this site"
// @Success      200  {object}  GoogleStartResponse
// @Failure      400  "invalid_next"
// @Failure      502  "provider_error"
// @Failure      503  "google_not_configured"
// @Router       /auth/google/start [get]
func (ctrl *Controller) GoogleStart(c *gin.Context) {
	start, err := ctrl.service.GoogleStart(origin(c), c.Query("next"))
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	// Lax, not Strict: the user comes back from Google through a cross-site
	// navigation and the callback page must still be able to present it.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SIGNIN_COOKIE, start.Binding, int(srv.SIGNIN_TTL.Seconds()), ctrl.signinPath(), "", ctrl.cookies.Secure, true)

	c.JSON(http.StatusOK, GoogleStartResponse{AuthorizationURL: start.URL})
}

// GoogleCallback godoc
// @Summary      Finish a Google sign-in and start a session
// @Description  Takes the code and state Google handed to the callback page. Only the browser that started the sign-in (tiu_signin cookie) can finish it, once.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  GoogleCallbackRequest  true  "What Google put in the callback URL"
// @Success      200  {object}  GoogleSessionResponse
// @Failure      400  "invalid_state"
// @Failure      403  "email_not_verified"
// @Failure      502  "provider_error"
// @Failure      503  "google_not_configured"
// @Router       /auth/google/callback [post]
func (ctrl *Controller) GoogleCallback(c *gin.Context) {
	var req GoogleCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	binding, _ := c.Cookie(SIGNIN_COOKIE)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SIGNIN_COOKIE, "", -1, ctrl.signinPath(), "", ctrl.cookies.Secure, true)

	tokens, next, err := ctrl.service.GoogleCallback(req.State, req.Code, binding)
	if err != nil {
		ctrl.fail(c, err)
		return
	}

	ctrl.setRefreshCookie(c, tokens)
	c.JSON(http.StatusOK, GoogleSessionResponse{SessionResponse: sessionResponse(tokens), Next: next})
}

func (ctrl *Controller) signinPath() string {
	return ctrl.cookies.Path + "/google"
}
