package service

import (
	"context"
)

// SuggestPoints отдаёт подсказку пунктов: уникальные from/to-пункты поездок
// и заявок с подстрочным поиском без учёта регистра. Тонкий делегат в
// репозиторий — доменных правил здесь нет (read-only проектор), слой service
// сохраняет принятую в проекте цепочку handler -> service -> repository.
func (s *TripService) SuggestPoints(ctx context.Context, term string) ([]string, error) {
	return s.tripRepository.SuggestPoints(ctx, term)
}
