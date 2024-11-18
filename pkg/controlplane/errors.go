package controlplane

type Entity string

func IsNotFoundErr(err error) bool {
	_, ok := err.(*ErrNotFound)
	return ok
}

func NewNotFoundErr(entity Entity) *ErrNotFound {
	return &ErrNotFound{
		entity: entity,
	}
}

type ErrNotFound struct {
	entity   Entity
	id       string
	parentId string
	err      error
}

func (e *ErrNotFound) Unwrap() error {
	return e.err
}

func (e *ErrNotFound) WithEntity(entity Entity) *ErrNotFound {
	e.entity = entity
	return e
}

func (e *ErrNotFound) WithId(id string) *ErrNotFound {
	e.id = id
	return e
}

func (e *ErrNotFound) WithParentId(id string) *ErrNotFound {
	e.parentId = id
	return e
}

func (e *ErrNotFound) WithErr(err error) *ErrNotFound {
	e.err = err
	return e
}

func (e ErrNotFound) Error() string {
	msg := string(e.entity)
	if e.id != "" {
		msg += " id:" + e.id
	}

	if e.parentId != "" {
		msg += " for:" + e.parentId
	}

	msg += " not found"

	if e.err != nil {
		msg += " error:" + e.err.Error()
	}
	return msg
}
