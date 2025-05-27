package repository

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"strings"
	"todo"
)

type TodoListPostgres struct {
	db *sqlx.DB
}

func NewTodoListPostgres(db *sqlx.DB) *TodoListPostgres {
	return &TodoListPostgres{db: db}
}

func (s *TodoListPostgres) Create(userId int, list todo.TodoList) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}

	var id int
	createListQuery := fmt.Sprintf(
		"INSERT INTO %s (title, description) VALUES ($1, $2) RETURNING id", todoListsTable)
	row := tx.QueryRow(createListQuery, list.Title, list.Description)
	if err := row.Scan(&id); err != nil {
		tx.Rollback()
		return 0, err
	}

	createUsersListQuery := fmt.Sprintf(
		"INSERT INTO %s (user_id, list_id) VALUES ($1, $2)", usersListsTable)
	_, err = tx.Exec(createUsersListQuery, userId, id)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, tx.Commit()
}

func (s *TodoListPostgres) GetAll(userId int) ([]todo.TodoList, error) {
	var lists []todo.TodoList

	query := fmt.Sprintf(
		"SELECT tl.id, tl.title, tl.description FROM %s tl "+
			"INNER JOIN %s ul on tl.id = ul.list_id "+
			"WHERE ul.user_id = $1",
		todoListsTable,
		usersListsTable)

	err := s.db.Select(&lists, query, userId)

	return lists, err
}

func (s *TodoListPostgres) GetById(userId, listId int) (todo.TodoList, error) {
	var list todo.TodoList

	query := fmt.Sprintf(
		"SELECT tl.id, tl.title, tl.description FROM %s tl "+
			"INNER JOIN %s ul on tl.id = ul.list_id "+
			"WHERE ul.user_id = $1 AND ul.list_id = $2",
		todoListsTable,
		usersListsTable)

	err := s.db.Get(&list, query, userId, listId)

	return list, err
}

func (s *TodoListPostgres) Delete(userId, listId int) error {
	query := fmt.Sprintf(
		"DELETE FROM %s tl USING %s ul WHERE tl.id = ul.list_id AND ul.user_id = $1 and ul.list_id = $2",
		todoListsTable, usersListsTable)
	_, err := s.db.Exec(query, userId, listId)

	return err
}

//func (s *TodoListPostgres) Delete(userId, listId int) error {
//	tx, err := s.db.Begin()
//	if err != nil {
//		return err
//	}
//
//	deleteRelationQuery := fmt.Sprintf(
//		"DELETE FROM %s WHERE user_id = $1 AND list_id = $2",
//		usersListsTable)
//	_, err = tx.Exec(deleteRelationQuery, userId, listId)
//	if err != nil {
//		tx.Rollback()
//		return err
//	}
//
//	deleteListQuery := fmt.Sprintf(
//		"DELETE FROM %s WHERE id = $1", todoListsTable)
//	_, err = tx.Exec(deleteListQuery, listId)
//	if err != nil {
//		tx.Rollback()
//		return err
//	}
//
//	return tx.Commit()
//}

func (s *TodoListPostgres) Update(userId, listId int, input todo.UpdateListInput) error {
	setValues := make([]string, 0)
	args := make([]interface{}, 0)
	argId := 1

	if input.Title != nil {
		setValues = append(setValues, fmt.Sprintf("title=$%d", argId))
		args = append(args, *input.Title)
		argId++
	}

	if input.Description != nil {
		setValues = append(setValues, fmt.Sprintf("description=$%d", argId))
		args = append(args, *input.Description)
		argId++
	}

	setQuery := strings.Join(setValues, ", ")

	query := fmt.Sprintf(
		"UPDATE %s tl SET %s "+
			"FROM %s ul WHERE tl.id = ul.list_id "+
			"AND ul.list_id = $%d AND ul.user_id = $%d",
		todoItemsTable,
		setQuery,
		usersListsTable,
		argId,
		argId+1)
	args = append(args, listId, userId)
	logrus.Infof("updateQuery: %s", query)
	logrus.Infof("args: %s", args)

	_, err := s.db.Exec(query, args...)
	return err
}
