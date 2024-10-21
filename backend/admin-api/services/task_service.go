package services

import (
	"admin-api/models"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TaskService struct {
	logger                    *otelzap.Logger
	taskRunArtifactRepository *models.TaskRunArtifactRepository
}

func NewTaskService(logger *otelzap.Logger, taskRunMetadataRepository *models.TaskRunArtifactRepository) *TaskService {
	return &TaskService{logger: logger, taskRunArtifactRepository: taskRunMetadataRepository}
}

func (s *TaskService) GetTasksByUserId(ctx context.Context, userId string) ([]models.TaskDto, error) {
	tasks, err := models.GetTasksByUserId(userId)
	if err != nil {
		s.logger.Ctx(ctx).Error("Failed find tasks", zap.Error(err))
		return nil, err
	}

	taskDtos := []models.TaskDto{}
	for _, task := range tasks {
		taskDto, err := s.MapTaskToDto(ctx, &task)
		if err != nil {
			s.logger.Ctx(ctx).Error("Error while mapping task to dto", zap.Error(err))
			return nil, err
		}
		taskDtos = append(taskDtos, *taskDto)
	}
	return taskDtos, nil
}

func (s *TaskService) GetTaskById(ctx context.Context, taskID string) (*models.TaskDto, error) {
	taskIDUint, err := strconv.ParseUint(taskID, 10, 64)
	if err != nil {
		s.logger.Ctx(ctx).Error("Failed to parse task id", zap.Error(err))
		return nil, err
	}

	j, err := models.GetTaskFromCache(ctx, taskIDUint)
	if err != nil {
		s.logger.Ctx(ctx).Error("Error while getting task from cache", zap.Error(err))
		return nil, err
	}
	if j != nil {
		return j, nil
	}

	task, err := models.GetTaskById(taskIDUint)
	if err != nil {
		s.logger.Ctx(ctx).Error("Error while getting task from db", zap.Error(err))
		return nil, err
	}

	taskDto, err := s.MapTaskToDto(ctx, task)
	if err != nil {
		s.logger.Ctx(ctx).Error("Error while mapping task to dto", zap.Error(err))
		return nil, err
	}

	if err := models.SetTaskCache(ctx, taskDto); err != nil {
		s.logger.Ctx(ctx).Error("Error while setting task in cache", zap.Error(err))
		return nil, err
	}

	return taskDto, nil
}

func (s *TaskService) CreateTask(ctx context.Context, task models.Task, userID string) (*models.Task, error) {
	createTask := models.Task{
		Owner:          userID,
		TaskDefinition: task.TaskDefinition,
		TaskName:       task.TaskName,
	}

	if err := models.CreateTask(createTask); err != nil {
		s.logger.Ctx(ctx).Error("Failed to create task", zap.Error(err))
		return nil, err
	}

	return &task, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, task models.Task, userID string, taskID string) (*models.Task, error) {
	taskIDUint, err := strconv.ParseUint(taskID, 10, 64)
	if err != nil {
		s.logger.Ctx(ctx).Error("Failed to parse task id", zap.Error(err))
		return nil, err
	}

	existingTask, err := models.GetTaskById(taskIDUint)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Ctx(ctx).Error("Task not found", zap.Uint64("task_id", taskIDUint))
			return nil, fmt.Errorf("task not found")
		}
		s.logger.Ctx(ctx).Error("Error while getting task from db", zap.Error(err))
		return nil, err
	}

	existingTask.TaskDefinition = task.TaskDefinition
	existingTask.TaskName = task.TaskName
	existingTask.UpdatedAt = time.Now()

	if err := models.UpdateTask(*existingTask); err != nil {
		s.logger.Ctx(ctx).Error("Failed to update task", zap.Error(err))
		return nil, err
	}

	return existingTask, nil
}

func (s *TaskService) ListTaskRuns(ctx context.Context, taskID string) ([]*models.TaskRunDto, error) {
	taskIDUint, err := strconv.ParseUint(taskID, 10, 64)
	if err != nil {
		s.logger.Ctx(ctx).Error("Failed to parse task id", zap.Error(err))
		return nil, err
	}

	taskRuns, err := models.ListRunsForTask(taskIDUint)
	if err != nil {
		s.logger.Ctx(ctx).Error("Error while getting task runs", zap.Error(err))
		return nil, err
	}

	taskRunsDto := []*models.TaskRunDto{}
	for _, taskRun := range taskRuns {
		taskRunsDto = append(taskRunsDto, s.MapTaskRunToDto(ctx, &taskRun))
	}

	return taskRunsDto, nil
}

func (s *TaskService) GetTaskRunArtifacts(ctx context.Context, taskRunID string, page int, pageSize int) ([]*models.TaskRunArtifactDto, error) {
	taskRunIDUint, err := strconv.ParseUint(taskRunID, 10, 64)
	if err != nil {
		s.logger.Ctx(ctx).Error("Failed to parse task id", zap.Error(err))
		return nil, err
	}

	taskRun, err := models.GetTaskRun(taskRunIDUint)
	if err != nil {
		s.logger.Ctx(ctx).Error("Error while getting task runs", zap.Error(err))
		return nil, err
	}

	airflowUUID, err := gocql.ParseUUID(taskRun.AirflowInstanceID)
	if err != nil {
		s.logger.Ctx(ctx).Error("Error parsing AirflowInstanceID to UUID", zap.Error(err))
		return nil, err
	}

	offset := (page - 1) * pageSize

	artifacts, err := s.taskRunArtifactRepository.ListArtifactsByTaskRunID(airflowUUID, pageSize, offset)
	if err != nil {
		s.logger.Ctx(ctx).Error("Error while getting task run metadata", zap.Error(err))
		return nil, err
	}

	artifactsDto := []*models.TaskRunArtifactDto{}
	for _, artifact := range artifacts {
		artifactsDto = append(artifactsDto, s.MapTaskRunArtifactToDto(ctx, artifact))
	}

	return artifactsDto, nil
}

func (s *TaskService) MapTaskToDto(ctx context.Context, task *models.Task) (*models.TaskDto, error) {
	taskRun, err := models.GetLatestRunForTask(uint64(task.ID))
	if err != nil {
		s.logger.Ctx(ctx).Error("Error while getting task run from db", zap.Error(err))
		return nil, err
	}

	status := models.TaskStatusPending
	if taskRun != nil {
		status = taskRun.Status
	}

	taskDto := &models.TaskDto{
		ID:             strconv.FormatUint(uint64(task.ID), 10),
		TaskName:       task.TaskName,
		TaskDefinition: string(task.TaskDefinition),
		Status:         status,
		Owner:          task.Owner,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
		DeletedAt:      task.DeletedAt.Time,
	}

	return taskDto, nil
}

func (s *TaskService) MapTaskRunToDto(ctx context.Context, taskRun *models.TaskRun) *models.TaskRunDto {
	taskRunDto := &models.TaskRunDto{
		TaskID:       strconv.FormatUint(uint64(taskRun.TaskID), 10),
		Status:       taskRun.Status,
		StartTime:    taskRun.StartTime,
		EndTime:      taskRun.EndTime,
		ErrorMessage: taskRun.ErrorMessage,
	}
	return taskRunDto
}

func (s *TaskService) MapTaskRunArtifactToDto(ctx context.Context, artifact *models.TaskRunArtifact) *models.TaskRunArtifactDto {
	taskRunArtifactDto := &models.TaskRunArtifactDto{
		AirflowInstanceID: artifact.AirflowInstanceID.String(),
		AirflowTaskID:     artifact.AirflowTaskID.String(),
		ArtifactID:        artifact.ArtifactID.String(),
		CreatedAt:         artifact.CreatedAt,
		ArtifactType:      artifact.ArtifactType,
		URL:               artifact.URL,
		ContentType:       artifact.ContentType,
		ContentLength:     artifact.ContentLength,
		StatusCode:        artifact.StatusCode,
		AdditionalData:    artifact.AdditionalData,
	}
	return taskRunArtifactDto
}
