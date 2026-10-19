// Package userctx переносит id аутентифицированного пользователя через запрос.
//
// id становится известен глубоко в цепочке (в auth-middleware), но читать его
// нужно и хендлерам ниже, и логирующему middleware выше. Значения контекста
// текут только вниз, поэтому стоящий выше логгер не увидит значение, положенное
// ниже в дочерний контекст. Чтобы перекинуть его вверх, логгер кладёт на входе
// изменяемый holder (NewContext), auth-middleware заполняет его (WithUserID), а
// хендлеры и логгер читают (UserIDFrom).
package userctx

import "context"

type holder struct {
	id  int64
	set bool
}

type ctxKey struct{}

// NewContext возвращает контекст с пустым holder-ом. Логирующий middleware зовёт
// его в начале цепочки, чтобы заполненный ниже id был виден, когда запрос
// раскручивается обратно вверх для записи лога.
func NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, &holder{})
}

// WithUserID записывает id в holder, уже лежащий в ctx (его положил NewContext),
// и возвращает ctx как есть. Если holder-а нет — например, auth-middleware
// работает без логгера, как в изолированных тестах — прикрепляет новый
// заполненный holder и возвращает производный контекст.
func WithUserID(ctx context.Context, id int64) context.Context {
	if h, ok := ctx.Value(ctxKey{}).(*holder); ok {
		h.id, h.set = id, true
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, &holder{id: id, set: true})
}

// UserIDFrom возвращает id пользователя и признак того, что он был установлен.
func UserIDFrom(ctx context.Context) (int64, bool) {
	h, ok := ctx.Value(ctxKey{}).(*holder)
	if !ok || !h.set {
		return 0, false
	}
	return h.id, true
}
