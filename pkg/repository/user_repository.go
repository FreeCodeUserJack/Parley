package repository

type UserRepositoryInterface interface{}

type userRepository struct{}

func NewUserRepository() UserRepositoryInterface {
	return &userRepository{}
}
