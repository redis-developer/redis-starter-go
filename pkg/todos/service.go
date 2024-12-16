package todos

import (
	"context"
  "fmt"
	"regexp"
)

type TodosService struct {
	repository Repository
}

func formatId(id string) (string, error) {
	matched, err := regexp.Match(`^todos:`, []byte(id))

  if matched {
    return id, err
  }

  return fmt.Sprintf("todos:%s", id), err
}

func (c TodosService) One(ctx context.Context, id string) (*Todo, error) {
  normalizedId, err := formatId(id)

  if err != nil {
    return nil, err
  }

	return c.repository.One(ctx, normalizedId)
}

func NewService(repository Repository) *TodosService {
	return &TodosService{
		repository: repository,
	}
}
