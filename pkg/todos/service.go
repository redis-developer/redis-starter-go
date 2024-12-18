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

  if err != nil {
    return id, fmt.Errorf("Failed to format id: %w", err)
  }

	if matched {
		return id, err
	}

	return fmt.Sprintf("todos:%s", id), err
}

func (c TodosService) All(ctx context.Context) (*Todos, error) {
	return c.repository.All(ctx)
}

func (c TodosService) Search(ctx context.Context, name string, status string) (*Todos, error) {
	return c.repository.Search(ctx, name, status)
}

func (c TodosService) One(ctx context.Context, id string) (*Todo, error) {
	normalizedId, err := formatId(id)

	if err != nil {
		return nil, err
	}

	return c.repository.One(ctx, normalizedId)
}

func (c TodosService) Create(ctx context.Context, id string, name string) (*Todo, error) {
	normalizedId, err := formatId(id)

	if err != nil {
		return nil, err
	}

	return c.repository.Create(ctx, normalizedId, name)
}

func (c TodosService) Update(ctx context.Context, id string, status string) (*Todo, error) {
	normalizedId, err := formatId(id)
	todoStatus, ok := TodoStatusMap[status]

	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("Invalid status %s", status)
	}

	return c.repository.Update(ctx, normalizedId, todoStatus)
}

func (c TodosService) Del(ctx context.Context, id string) error {
	normalizedId, err := formatId(id)

	if err != nil {
		return err
	}

	return c.repository.Del(ctx, normalizedId)
}

func (c TodosService) DelAll(ctx context.Context) error {
	return c.repository.DelAll(ctx)
}

func NewService(repository Repository) *TodosService {
	return &TodosService{
		repository: repository,
	}
}
