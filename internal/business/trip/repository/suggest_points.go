package repository

import (
	"context"
	"fmt"
	"strings"
)

// suggestPointsLimit — максимум пунктов в одном ответе подсказки.
const suggestPointsLimit = 10

// suggestPointsQuery выбирает уникальные пункты из всех колонок поездок и
// заявок. WHERE использует ILIKE с параметром-шаблоном (паттерн собирается
// в escapeILIKE), сортировка по lower(point) даёт стабильный алфавитный
// порядок, лимит — в SQL, а не в памяти. DISTINCT живёт во внутреннем
// подзапросе: PostgreSQL запрещает ORDER BY выражением, отсутствующим в
// select list, при DISTINCT (SQLSTATE 42P10), поэтому сортировка и лимит
// вынесены наружу.
const suggestPointsQuery = `
		SELECT point FROM (
			SELECT DISTINCT point FROM (
				SELECT from_point AS point FROM public.trips
				UNION ALL
				SELECT to_point FROM public.trips
				UNION ALL
				SELECT from_point FROM public.trip_requests
				UNION ALL
				SELECT to_point FROM public.trip_requests
			) all_points
			WHERE point ILIKE $1
		) unique_points
		ORDER BY lower(point)
		LIMIT %d`

// SuggestPoints возвращает до suggestPointsLimit уникальных пунктов, в
// которых term встречается как подстрока без учёта регистра. Пустой term
// означает отсутствие фильтра: шаблон складывается в "%%", который матчит
// любую строку. Результат всегда не-nil срез — пустой ответ сериализуется
// в [], а не в null.
func (r *TripRepository) SuggestPoints(ctx context.Context, term string) ([]string, error) {
	pattern := "%" + escapeILIKE(term) + "%"
	query := fmt.Sprintf(suggestPointsQuery, suggestPointsLimit)

	rows, err := r.db.Query(ctx, query, pattern)
	if err != nil {
		return nil, fmt.Errorf("suggest points: %w", err)
	}
	defer rows.Close()

	points := make([]string, 0)
	for rows.Next() {
		var point string
		if err := rows.Scan(&point); err != nil {
			return nil, fmt.Errorf("scan point: %w", err)
		}
		points = append(points, point)
	}

	return points, rows.Err()
}

// escapeILIKE экранирует служебные символы шаблона ILIKE, чтобы %, _ и \
// из пользовательского ввода воспринимались буквально. Replacer работает за
// один проход, поэтому обратные слэши не удваиваются повторно.
func escapeILIKE(term string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(term)
}
