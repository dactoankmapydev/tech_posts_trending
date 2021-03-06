package handler

import (
	"devread/custom_error"
	"devread/helper"
	"devread/log"
	"devread/model"
	"devread/model/req"
	"devread/repository"
	"devread/security"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/smtp"
	"os"
)

type smtpServer struct {
	host string
	port string
}

// Address URI to smtp server.
func (s *smtpServer) Address() string {
	return s.host + ":" + s.port
}

type UserHandler struct {
	UserRepo repository.UserRepo
	AuthRepo repository.AuthenRepo
}

// SignUp godoc
// @Summary Create new account
// @Tags user-service
// @Accept  json
// @Produce  json
// @Param data body req.ReqSignUp true "user"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Failure 409 {object} model.Response
// @Router /user/sign-up [post]
func (u *UserHandler) SignUp(c echo.Context) error {
	request := req.ReqSignUp{}
	if err := c.Bind(&request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	if err := c.Validate(request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	hash := security.HashAndSalt([]byte(request.Password))

	userID, err := uuid.NewUUID()
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    err.Error(),
		})
	}

	user := model.User{
		UserID:   userID.String(),
		FullName: request.FullName,
		Email:    request.Email,
		Password: hash,
		Verify:   false,
	}

	user, err = u.UserRepo.SaveUser(c.Request().Context(), user)
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusConflict, model.Response{
			StatusCode: http.StatusConflict,
			Message:    err.Error(),
		})
	}

	// verify email
	token := helper.CreateTokenHash(user.Email)

	// save token to redis
	saveErr := u.AuthRepo.CreateTokenMail(token, user.UserID)
	if saveErr != nil {
		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    saveErr.Error(),
		})
	}

	link := "http://localhost:3000" + "/user/verify?token=" + token

	from := os.Getenv("FROM")
	password := os.Getenv("PASSWORD")
	to := []string{user.Email}

	smtpsv := smtpServer{
		host: os.Getenv("SMTP_HOST"),
		port: os.Getenv("SMTP_PORT"),
	}

	subject := "Xác thực tài khoản\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := "Để xác thực tài khoản nhấp vào liên kết <a href='" + link + "'>ở đây</a>."
	message := []byte("Subject:" + subject + mime + "\r\n" + body)

	auth := smtp.PlainAuth("", from, password, smtpsv.host)
	errSendMail := smtp.SendMail(smtpsv.Address(), auth, from, to, message)
	if errSendMail != nil {
		log.Error(errSendMail)
		return (errSendMail)
	}

	return c.JSON(http.StatusOK, model.Response{
		StatusCode: http.StatusOK,
		Message:    "Tin nhắn xác thực tài khoản được gửi đến email được cung cấp. Vui lòng kiểm tra thư mục thư rác",
	})
}

// ForgotPassword godoc
// @Summary Forgot password
// @Tags user-service
// @Accept  json
// @Produce  json
// @Param data body req.ReqEmail true "user"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /user/password/forgot [post]
func (u *UserHandler) ForgotPassword(c echo.Context) error {
	request := req.ReqEmail{}
	if err := c.Bind(&request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	if err := c.Validate(request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	user, err := u.UserRepo.CheckEmail(c.Request().Context(), request)
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    err.Error(),
		})
	}

	token := helper.CreateTokenHash(user.Email)

	// save token to redis
	saveErr := u.AuthRepo.CreateTokenMail(token, user.UserID)
	if saveErr != nil {
		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    saveErr.Error(),
		})
	}

	insertErr := u.AuthRepo.InsertTokenMail(token)
	if insertErr != nil {
		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    insertErr.Error(),
		})
	}

	link := "http://localhost:3000" + "/user/password/reset?token=" + token

	from := os.Getenv("FROM")
	password := os.Getenv("PASSWORD")
	to := []string{user.Email}

	smtpsv := smtpServer{
		host: os.Getenv("SMTP_HOST"),
		port: os.Getenv("SMTP_PORT"),
	}

	subject := "Đặt lại mật khẩu\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := "Để đặt lại mật khẩu nhấp vào liên kết <a href='" + link + "'>ở đây</a>."
	message := []byte("Subject:" + subject + mime + "\r\n" + body)

	auth := smtp.PlainAuth("", from, password, smtpsv.host)
	errSendMail := smtp.SendMail(smtpsv.Address(), auth, from, to, message)
	if errSendMail != nil {
		return errSendMail
	}

	return c.JSON(http.StatusOK, model.Response{
		StatusCode: http.StatusOK,
		Message:    "Tin nhắn được gửi đến email được cung cấp. Vui lòng kiểm tra thư mục thư rác",
	})
}

// VerifyAccount ... godoc
// @Summary Verify email
// @Tags user-service
// @Accept  json
// @Produce  json
// @Param data body req.PasswordSubmit true "user"
// @Param token query string true "token verify email"
// @Security token-verify-account
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /user/verify [post]
func (u *UserHandler) VerifyAccount(c echo.Context) error {
	request := req.PasswordSubmit{}
	if err := c.Bind(&request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	if err := c.Validate(request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	token := security.ExtractTokenMail(c.Request())

	userID, err := u.AuthRepo.FetchTokenMail(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Truy cập không được phép",
		})
	}

	user, err := u.UserRepo.SelectUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    err.Error(),
		})
	}

	if request.Password != request.Confirm {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Xác nhận mật khẩu không khớp",
		})
	}

	// check password
	isTheSame := security.ComparePasswords(user.Password, []byte(request.Password))
	if !isTheSame {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Mật khẩu không đúng",
		})
	}

	user = model.User{
		UserID: userID,
		Verify: true,
	}

	user, err = u.UserRepo.UpdateVerify(c.Request().Context(), user)
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    err.Error(),
		})
	}

	deleteAtErr := u.AuthRepo.DeleteTokenMail(token)
	if deleteAtErr != nil {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    deleteAtErr.Error(),
		})
	}

	return c.JSON(http.StatusOK, model.Response{
		StatusCode: http.StatusOK,
		Message:    "Xác thực tài khoản thành công",
	})

}

// ResetPassword godoc
// @Summary Reset password
// @Tags user-service
// @Accept  json
// @Produce  json
// @Param data body req.PasswordSubmit true "user"
// @Param token query string true "token reset password"
// @Security token-reset-password
// @Success 201 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /user/password/reset [put]
func (u *UserHandler) ResetPassword(c echo.Context) error {
	request := req.PasswordSubmit{}
	if err := c.Bind(&request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	if err := c.Validate(request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	token := security.ExtractTokenMail(c.Request())

	userID, err := u.AuthRepo.FetchTokenMail(token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Truy cập không được phép, cần gửi lại email",
		})
	}

	if request.Password != request.Confirm {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Xác nhận mật khẩu không khớp",
		})
	}

	hash := security.HashAndSalt([]byte(request.Password))

	user := model.User{
		UserID:   userID,
		Password: hash,
	}

	user, err = u.UserRepo.UpdatePassword(c.Request().Context(), user)
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    err.Error(),
		})
	}

	deleteAtErr := u.AuthRepo.DeleteTokenMail(token)
	if deleteAtErr != nil {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    deleteAtErr.Error(),
		})
	}

	return c.JSON(http.StatusCreated, model.Response{
		StatusCode: http.StatusCreated,
		Message:    "Tạo mới mật khẩu thành công",
	})
}

// SignIn godoc
// @Summary Sign in to access your account
// @Tags user-service
// @Accept  json
// @Produce  json
// @Param data body req.ReqSignIn true "user"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /user/sign-in [post]
func (u *UserHandler) SignIn(c echo.Context) error {
	request := req.ReqSignIn{}
	if err := c.Bind(&request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	if err := c.Validate(request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	user, err := u.UserRepo.CheckSignIn(c.Request().Context(), request)
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    err.Error(),
		})
	}

	if user.Verify != true {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Tài khoản chưa được xác thực",
		})
	}

	// check password
	isTheSame := security.ComparePasswords(user.Password, []byte(request.Password))
	if !isTheSame {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Email hoặc mật khẩu không chính xác",
		})
	}

	// create token
	token, err := security.CreateToken(user)
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    err.Error(),
		})
	}
	user.Token = token

	return c.JSON(http.StatusOK, model.Response{
		StatusCode: http.StatusOK,
		Message:    "Đăng nhập thành công",
		Data:       user,
	})
}

// Profile godoc
// @Summary Get user profile
// @Tags profile-service
// @Accept  json
// @Produce  json
// @Security jwt
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /user/profile [get]
func (u *UserHandler) Profile(c echo.Context) error {
	tokenData := c.Get("user").(*jwt.Token)
	claims := tokenData.Claims.(*model.TokenDetails)

	user, err := u.UserRepo.SelectUserByID(c.Request().Context(), claims.UserID)
	if err != nil {
		log.Error(err.Error())
		if err == custom_error.UserNotFound {
			return c.JSON(http.StatusNotFound, model.Response{
				StatusCode: http.StatusNotFound,
				Message:    err.Error(),
			})
		}

		return c.JSON(http.StatusForbidden, model.Response{
			StatusCode: http.StatusForbidden,
			Message:    err.Error(),
		})
	}

	return c.JSON(http.StatusOK, model.Response{
		StatusCode: http.StatusOK,
		Message:    "Xử lý thành công",
		Data:       user,
	})
}

// UpdateProfile godoc
// @Summary Update user profile
// @Tags profile-service
// @Accept  json
// @Produce  json
// @Security jwt
// @Param data body req.ReqUpdateUser true "user"
// @Success 201 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 422 {object} model.Response
// @Router /user/profile/update [put]
func (u *UserHandler) UpdateProfile(c echo.Context) error {
	request := req.ReqUpdateUser{}
	if err := c.Bind(&request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	if err := c.Validate(request); err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusBadRequest, model.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "Lỗi cú pháp",
		})
	}

	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*model.TokenDetails)

	if request.Password != request.Confirm {
		return c.JSON(http.StatusUnauthorized, model.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "Xác nhận mật khẩu không khớp",
		})
	}

	hash := security.HashAndSalt([]byte(request.Password))

	var err error
	if request.FullName == "" {
		if len(request.Password) < 8 {
			return c.JSON(http.StatusBadRequest, model.Response{
				StatusCode: http.StatusBadRequest,
				Message:    "Mật khẩu tối thiểu 8 ký tự",
			})
		}
		user := model.User{
			UserID:   claims.UserID,
			Password: hash,
		}

		user, err = u.UserRepo.UpdateUser(c.Request().Context(), user)
		if err != nil {
			log.Error(err.Error())
			return c.JSON(http.StatusUnprocessableEntity, model.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Message:    err.Error(),
			})
		}
	}

	if request.Password == "" {
		user := model.User{
			UserID:   claims.UserID,
			FullName: request.FullName,
		}

		user, err = u.UserRepo.UpdateUser(c.Request().Context(), user)
		if err != nil {
			log.Error(err.Error())
			return c.JSON(http.StatusUnprocessableEntity, model.Response{
				StatusCode: http.StatusUnprocessableEntity,
				Message:    err.Error(),
			})
		}
	}

	user := model.User{
		UserID:   claims.UserID,
		FullName: request.FullName,
		Password: hash,
	}

	user, err = u.UserRepo.UpdateUser(c.Request().Context(), user)
	if err != nil {
		log.Error(err.Error())
		return c.JSON(http.StatusUnprocessableEntity, model.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Message:    err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, model.Response{
		StatusCode: http.StatusCreated,
		Message:    "Cập nhật thông tin thành công",
	})
}