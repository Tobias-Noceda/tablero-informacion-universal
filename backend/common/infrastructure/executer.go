package infrastructure

import "github.com/Secreto31126/tesis/common/models"

type Executer interface {
	Execute(postit *models.PostIts) (any, error)
}
