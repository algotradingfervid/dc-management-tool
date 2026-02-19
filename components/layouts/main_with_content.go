package layouts

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// MainWithContent returns a templ.Component that renders the Main layout with
// the given content component as its children. Use this from Go handler code
// where you cannot use templ's @Component{} child-passing syntax directly.
func MainWithContent(
	title string,
	sidebar templ.Component,
	topbar templ.Component,
	flashMessage string,
	flashType string,
	content templ.Component,
) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Main(title, sidebar, topbar, flashMessage, flashType).Render(
			templ.WithChildren(ctx, content),
			w,
		)
	})
}
