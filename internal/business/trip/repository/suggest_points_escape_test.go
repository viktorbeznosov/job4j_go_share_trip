package repository

import (
	"strings"
	"testing"
)

// matchLike эмулирует семантику LIKE/ILIKE для проверки шаблонов без
// поднятия БД: % — любая последовательность символов, _ — ровно один
// символ, \ — литерализует следующий за ним символ. Сравнение без учёта
// регистра, как у ILIKE.
func matchLike(s, pattern string) bool {
	return matchLikeRec(
		[]rune(strings.ToLower(s)),
		[]rune(strings.ToLower(pattern)),
		0,
		0,
	)
}

func matchLikeRec(s, p []rune, si, pi int) bool {
	for pi < len(p) {
		switch p[pi] {
		case '\\':
			if pi+1 >= len(p) {
				return false
			}
			if si >= len(s) || s[si] != p[pi+1] {
				return false
			}
			si++
			pi += 2
		case '%':
			// Пробуем съесть 0..len(s)-1 символов.
			for i := si; i <= len(s); i++ {
				if matchLikeRec(s, p, i, pi+1) {
					return true
				}
			}
			return false
		case '_':
			if si >= len(s) {
				return false
			}
			si++
			pi++
		default:
			if si >= len(s) || s[si] != p[pi] {
				return false
			}
			si++
			pi++
		}
	}
	return si == len(s)
}

func TestEscapeILIKE(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"обычный текст без изменений", "москва", "москва"},
		{"процент экранируется", "100%", `100\%`},
		{"подчёркивание экранируется", "м_сква", `м\_сква`},
		{"обратный слэш экранируется", `C:\path`, `C:\\path`},
		{"все спецсимволы сразу", `a%b_c\d`, `a\%b\_c\\d`},
		{"пустая строка", "", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := escapeILIKE(tc.in); got != tc.want {
				t.Errorf("escapeILIKE(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestEscapeILIKE_PercentDoesNotMatchEverything проверяет, что собранный
// из пользовательского "%" шаблон матчит только литеральный процент, а не
// любую строку — регрессия к сценарию "Процент в запросе не матчит всё".
func TestEscapeILIKE_PercentDoesNotMatchEverything(t *testing.T) {
	t.Parallel()

	// Шаблон строится как % + escapeILIKE(term) + %, как в SuggestPoints.
	pattern := "%" + escapeILIKE("%") + "%"
	if pattern != `%\%%` {
		t.Fatalf("pattern = %q, want %q", pattern, `%\%%`)
	}

	// Литеральный % в шаблоне экранирован, поэтому "матчит всё" не происходит.
	if !matchLike("100% скидка", pattern) {
		t.Errorf("pattern %q should match a string containing a literal %%"+"%%", pattern)
	}
	if matchLike("москва", pattern) {
		t.Errorf("pattern %q must not match a string without a percent sign", pattern)
	}
}

// TestEscapeILIKE_UnderscoreDoesNotMatchArbitraryChar проверяет, что "_" из
// ввода не работает как wildcard: "м_сква" не должно находить "москва".
func TestEscapeILIKE_UnderscoreDoesNotMatchArbitraryChar(t *testing.T) {
	t.Parallel()

	pattern := "%" + escapeILIKE("м_сква") + "%"
	if matchLike("москва", pattern) {
		t.Errorf("pattern %q must not match \"москва\": _ is literal, not a wildcard", pattern)
	}
	if !matchLike("м_сква", pattern) {
		t.Errorf("pattern %q should match the literal string it was built from", pattern)
	}
}
