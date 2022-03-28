package auth

import (
	"net/http"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/convert"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/password"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/mailer"
)

// KitspaceSignUp custom sign-up compatible with Kitspace architecture
func KitspaceSignUp(ctx *context.Context) {
	// swagger:operation POST /user/kitspace/sign_up
	// ---
	// summary: Create a user
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/RegisterForm"
	// responses:
	//   "201":
	//     "$ref": "#/responses/User"
	//   "400":
	//     "$ref": "#/responses/error"
	//	 "409":
	//     "$ref": "#/response/error
	//   "422":
	//     "$ref": "#/responses/validationError"
	response := make(map[string]interface{})
	form := web.GetForm(ctx).(*forms.RegisterForm)

	if len(form.Password) < setting.MinPasswordLength {
		response["error"] = "UnprocessableEntity"
		response["message"] = "Password is too short."

		ctx.JSON(http.StatusUnprocessableEntity, response)
		return
	}

	if !password.IsComplexEnough(form.Password) {
		response["error"] = "UnprocessableEntity"
		response["message"] = "Password isn't complex enough."

		ctx.JSON(http.StatusUnprocessableEntity, response)
		return
	}

	u := &user_model.User{
		Name:     form.UserName,
		Email:    form.Email,
		Passwd:   form.Password,
		IsActive: !setting.Service.RegisterEmailConfirm,
	}

	if err := user_model.CreateUser(u); err != nil {
		switch {
		case user_model.IsErrUserAlreadyExist(err):
			response["error"] = "Conflict"
			response["message"] = "User already exists."

			ctx.JSON(http.StatusConflict, response)
		case user_model.IsErrEmailAlreadyUsed(err):
			response["error"] = "Conflict"
			response["message"] = "Email is already used."

			ctx.JSON(http.StatusConflict, response)
		case db.IsErrNameReserved(err):
			response["error"] = "Conflict"
			response["message"] = "Name is reserved."

			ctx.JSON(http.StatusConflict, response)
		case db.IsErrNamePatternNotAllowed(err):
			response["error"] = "UnprocessableEntity"
			response["message"] = "This name pattern isn't allowed."

			ctx.JSON(http.StatusUnprocessableEntity, response)
		default:
			ctx.ServerError("Signup", err)
		}
		return
	} else {
		log.Trace("Account created: %s", u.Name)
	}

	// Send confirmation email
	// The mailing service works only in production during development no mails are sent
	if setting.Service.RegisterEmailConfirm && u.ID > 1 {
		mailer.SendActivateAccountMail(ctx.Locale, u)

		if err := ctx.Cache.Put("MailResendLimit_"+u.LowerName, u.LowerName, 180); err != nil {
			log.Error("Set cache(MailResendLimit) fail: %v", err)
		}
	}

	handleSignInFull(ctx, u, true, false)

	// Return the success response with user details
	response["user"] = convert.ToUser(u, u)

	ctx.JSON(http.StatusCreated, response)
}

// KitspaceSignIn custom sign-in compatible with Kitspace architecture
func KitspaceSignIn(ctx *context.Context) {
	// swagger:operation POST /user/kitspace/sign_in
	// ---
	// summary: login a user
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/SignInForm"
	// responses:
	//   "200":
	//     "$ref": "success"
	//   "404":
	//     "$ref": "#/response/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//	 "409":
	//     "$ref": "#/response/error
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*forms.SignInForm)
	u, _, err := auth.UserSignIn(form.UserName, form.Password)

	response := make(map[string]interface{})
	if err != nil {
		switch {
		case user_model.IsErrUserNotExist(err):
			response["error"] = "Not Found"
			response["message"] = "Wrong username or password."

			ctx.JSON(http.StatusNotFound, response)
			log.Info("Failed authentication attempt for %s from %s", form.UserName, ctx.RemoteAddr())
		case user_model.IsErrEmailAlreadyUsed(err):
			response["error"] = "Conflict"
			response["message"] = "This email has already been used."

			ctx.JSON(http.StatusConflict, response)
			log.Info("Failed authentication attempt for %s from %s", form.UserName, ctx.RemoteAddr())
		case user_model.IsErrUserProhibitLogin(err):
			response["error"] = "Prohibited"
			response["message"] = "Prohibited login."

			ctx.JSON(http.StatusForbidden, response)
			log.Info("Failed authentication attempt for %s from %s", form.UserName, ctx.RemoteAddr())
		case user_model.IsErrUserInactive(err):
			if setting.Service.RegisterEmailConfirm {
				response["error"] = "ActivationRequired"
				response["message"] = "Activate your account."

				ctx.JSON(http.StatusOK, response)
			} else {
				response["error"] = "Prohibited"
				response["message"] = "Prohibited login"

				ctx.JSON(http.StatusForbidden, response)
				log.Info("Failed authentication attempt for %s from %s", form.UserName, ctx.RemoteAddr())
			}
		default:
			ctx.ServerError("KitspaceSignIn", err)
		}
		return
	}
	handleSignInFull(ctx, u, form.Remember, false)

	response["user"] = convert.ToUser(u, u)

	ctx.JSON(http.StatusOK, response)
}

func GetKitspaceSession(ctx *context.Context) {
	// swagger:operation GET /user/kitspace/session
	// ---
	// summary: get currently signed in user (if any) and csrf token
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "success"

	var user *structs.User
	if ctx.Doer != nil && ctx.IsSigned {
		user = convert.ToUser(ctx.Doer, ctx.Doer)
	}

	response := make(map[string]interface{})
	response["user"] = user
	response["csrf"] = ctx.Data["CsrfToken"]

	ctx.JSON(http.StatusOK, response)
}
