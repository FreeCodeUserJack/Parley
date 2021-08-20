package services

import "github.com/FreeCodeUserJack/Parley/pkg/repository"

type UserServiceInterface interface{}

type userService struct {
	UserRepository repository.UserRepositoryInterface
}

func NewUserService(userRepo repository.UserRepositoryInterface) UserServiceInterface {
	return userService{
		UserRepository: userRepo,
	}
}
