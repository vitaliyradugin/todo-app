package repository

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"strings"
	"todo"
)

type TodoItemPostgres struct {
	db *sqlx.DB
}

func NewTodoItemPostgres(db *sqlx.DB) *TodoItemPostgres {
	return &TodoItemPostgres{db: db}
}

func (s *TodoItemPostgres) Create(listId int, item todo.TodoItem) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}

	var itemId int
	createItemQuery := fmt.Sprintf(
		"INSERT INTO %s (title, description) VALUES ($1, $2) RETURNING id",
		todoItemsTable,
	)
	row := tx.QueryRow(createItemQuery, item.Title, item.Description)
	err = row.Scan(&itemId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	createListItemsQuery := fmt.Sprintf(
		"INSERT INTO %s (list_id, item_id) VALUES ($1, $2)",
		listsItemsTable,
	)
	_, err = tx.Exec(createListItemsQuery, listId, itemId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return itemId, tx.Commit()
}

func (s *TodoItemPostgres) GetAll(userId, listId int) ([]todo.TodoItem, error) {
	var items []todo.TodoItem
	query := fmt.Sprintf(
		"SELECT ti.title, ti.description, ti.done FROM %s ti "+
			"INNER JOIN %s li ON ti.id = li.item_id "+
			"INNER JOIN %s ul ON ul.list_id = li.list_id "+
			"WHERE li.list_id = $1 AND ul.user_id = $2",
		todoItemsTable,
		listsItemsTable,
		usersListsTable,
	)

	if err := s.db.Select(&items, query, listId, userId); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *TodoItemPostgres) GetById(userId, itemId int) (todo.TodoItem, error) {
	var item todo.TodoItem
	query := fmt.Sprintf(
		"SELECT ti.id, ti.title, ti.description, ti.done FROM %s ti "+
			"INNER JOIN %s li ON ti.id = li.item_id "+
			"INNER JOIN %s ul ON ul.list_id = li.list_id "+
			"WHERE ti.id = $1 AND ul.user_id = $2",
		todoItemsTable,
		listsItemsTable,
		usersListsTable,
	)

	if err := s.db.Get(&item, query, itemId, userId); err != nil {
		return item, err
	}

	return item, nil
}

func (s *TodoItemPostgres) Update(userId, itemId int, input todo.UpdateItemInput) error {
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

	if input.Done != nil {
		setValues = append(setValues, fmt.Sprintf("done=$%d", argId))
		args = append(args, *input.Done)
		argId++
	}

	setQuery := strings.Join(setValues, ", ")

	query := fmt.Sprintf(
		"UPDATE %s ti SET %s "+
			"FROM %s li, %s ul "+
			"WHERE li.item_id = ti.id "+
			"AND ul.list_id = li.list_id "+
			"AND ul.user_id=$%d and ti.id=$%d",
		todoItemsTable,
		setQuery,
		listsItemsTable,
		usersListsTable,
		argId,
		argId+1,
	)
	args = append(args, itemId, userId)
	logrus.Infof("updateQuery: %s", query)
	logrus.Infof("args: %s", args)

	_, err := s.db.Exec(query, args...)
	return err
}

func (s *TodoItemPostgres) Delete(userId, itemId int) error {
	query := fmt.Sprintf(
		"DELETE FROM %s ti "+
			"USING %s li, %s ul "+
			"WHERE ti.id = li.item_id AND ul.list_id = li.list_id "+
			"AND ul.user_id = $1 and ti.id = $2",
		todoItemsTable, listsItemsTable, usersListsTable)
	res, err := s.db.Exec(query, userId, itemId)

	logrus.Infof("deleteQuery: %s", query)
	logrus.Infof("res: %s", res)
	return err
}
