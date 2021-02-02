package fn

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DecoratorTestSuite struct {
	suite.Suite
}

func (suite *DecoratorTestSuite) SetupTest() {
}

func TestDecoratorTestSuite(t *testing.T) {
	suite.Run(t, new(DecoratorTestSuite))
}

type Handler1 func(int) string

type Handler2 func(int) string

type Middleware1 func(Handler1) Handler1

func (suite *DecoratorTestSuite) TestDecorateFunc() {
	origFn := func(i int) string { return fmt.Sprint(i) }
	var outFn func(int) string
	DecoratorOf(origFn).AddMW(func(h Handler1) Handler1 {
		return func(i int) string {
			return h(i + 1)
		}
	}).AddMW(func(h func(int) string) func(int) string {
		return func(i int) string {
			return h(i + 1)
		}
	}).AddMW(func(h Handler2) Handler2 {
		return func(i int) string {
			return h(i + 1)
		}
	}).Export(&outFn)

	suite.Equal("4", outFn(1))
}

func (suite *DecoratorTestSuite) TestPrependFunc() {
	origFn := func(i int) string { return fmt.Sprint(i) }
	var outFn Handler2
	DecoratorOf(origFn).AddMW(func(h Handler1) Handler1 {
		return func(i int) string {
			return h(i) + "A"
		}
	}).PrependMW(func(h func(int) string) func(int) string {
		return func(i int) string {
			return h(i) + "B"
		}
	}).InsertMW(1, func(h Handler2) Handler1 {
		return func(i int) string {
			return h(i) + "C"
		}
	}).Export(&outFn)

	suite.Equal("1BCA", outFn(1))
}
