package user

import (
	"context"

	"github.com/JadnaSantos/Gobid.git/internal/validator"
)

type SignUserReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req SignUserReq) Valid(ctx context.Context) validator.Evaluator {
	var eval validator.Evaluator

	eval.CheckField(validator.Matches(req.Email, validator.EmailRX), "email", "must be a valid email")
	eval.CheckField(validator.NotBlank(req.Password), "password", "this field cannot be blank")

	return eval
}
