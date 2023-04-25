package xeger

import (
	"fmt"
	"math/rand"
	"regexp"
	"regexp/syntax"
	"strconv"
	"time"
)

const (
	ascii_lowercase = "abcdefghijklmnopqrstuvwxyz"
	ascii_uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ascii_letters   = ascii_lowercase + ascii_uppercase
	digits          = "0123456789"
	punctuation     = " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	control         = "\t\v\f\r"
	newline         = "\n"
	printable       = digits + ascii_letters + punctuation + control + newline
	printableNotNL  = digits + ascii_letters + punctuation + control
	printableNotControl = digits + ascii_letters + punctuation + newline
)

var src = rand.NewSource(time.Now().UnixNano())

const limit = 10

type Xeger struct {
	re *syntax.Regexp
	groups map[int]string
}

func NewXeger(regex string) (*Xeger, error) {
	re, err := syntax.Parse(regex, syntax.Perl)
	if err != nil {
		return nil, err
	}

	x := &Xeger{re: re, groups: make(map[int]string)}
	return x, nil
}

func (x *Xeger) Generate() string {
	return x.generateFromRegexp(x.re)
}

// Generates strings which are matched with re.
func (x *Xeger) generateFromRegexp(re *syntax.Regexp) string {
	switch re.Op {
	case syntax.OpLiteral: // matches Runes sequence
		// Строка из re.Rune
		s := string(re.Rune)

		// Регулярное выражение для проверки строки вида <g\d+>
		rgx, err := regexp.Compile(`<g(\d+)>`)
		if err != nil {
			panic(fmt.Sprintf("ошибка при компиляции регулярного выражения: %v", err))
		}

		// Если строка соответствует шаблону <g\d+>
		if rgx.MatchString(s) {
			// Найти индекс группы
			matches := rgx.FindStringSubmatch(s)
			index, err := strconv.Atoi(matches[1])
			if err != nil {
				panic(fmt.Sprintf("ошибка при преобразовании индекса группы: %v", err))
			}

			// Заменить строку значением из массива groups по индексу \d+
			if val, ok := x.groups[index]; ok {
				return val
			} else {
				// Если \d+ больше чем элементов в groups, заменить на octal \d+
				if index > len(x.groups) {
					octal, err := strconv.ParseInt(matches[1], 8, 32)
					if err != nil {
						panic(fmt.Sprintf("ошибка при преобразовании octal: %v", err))
					}
					return string(rune(octal))
				} else {
					return ""
				}
			}
		}

		return s

	case syntax.OpCharClass: // matches Runes interpreted as range pair list
		var filtered []rune
		if re.Flags&syntax.FoldCase != 0 {
			// If it is a negated character class
			for _, r := range printableNotControl {
				if !isInRanges(r, re.Rune) {
					filtered = append(filtered, r)
				}
			}
		} else {
			// If it is a regular character class
			for _, r := range printableNotControl {
				if isInRanges(r, re.Rune) {
					filtered = append(filtered, r)
				}
			}
		}

		if len(filtered) > 0 {
			candidate := filtered[randInt(len(filtered))]
			return string(candidate)
		}
		return ""

	case syntax.OpAnyCharNotNL: // matches any character except newline
		c := printableNotNL[randInt(len(printableNotNL))]
		return string([]byte{c})

	case syntax.OpAnyChar: // matches any character
		c := printable[randInt(len(printable))]
		return string([]byte{c})

	case syntax.OpCapture: // capturing subexpression with index Cap, optional name Name
		generated := x.generateFromSubexpression(re, 1)
		x.groups[re.Cap] = generated
		return generated

	case syntax.OpStar: // matches Sub[0] zero or more times
		return x.generateFromSubexpression(re, randInt(limit+1))

	case syntax.OpPlus: // matches Sub[0] one or more times
		return x.generateFromSubexpression(re, randInt(limit)+1)

	case syntax.OpQuest: // matches Sub[0] zero or one times
		return x.generateFromSubexpression(re, randInt(2))

	case syntax.OpRepeat: // matches Sub[0] at least Min times, at most Max (Max == -1 is no limit)
		max := re.Max
		if max == -1 {
			max = limit
		}
		count := randInt(max-re.Min+1) + re.Min
		return x.generateFromSubexpression(re, count)

	case syntax.OpConcat: // matches concatenation of Subs
		return x.generateFromSubexpression(re, 1)

	case syntax.OpAlternate: // matches alternation of Subs
		i := randInt(len(re.Sub))
		return x.generateFromRegexp(re.Sub[i])

		/*
			// The other cases return empty string.
			case syntax.OpNoMatch: // matches no strings
			case syntax.OpEmptyMatch: // matches empty string
			case syntax.OpBeginLine: // matches empty string at beginning of line
			case syntax.OpEndLine: // matches empty string at end of line
			case syntax.OpBeginText: // matches empty string at beginning of text
			case syntax.OpEndText: // matches empty string at end of text
			case syntax.OpWordBoundary: // matches word boundary `\b`
			case syntax.OpNoWordBoundary: // matches word non-boundary `\B`
		*/
	}

	return ""
}

// Generates strings from all sub-expressions.
// If count > 1, repeat to generate.
func (x *Xeger) generateFromSubexpression(re *syntax.Regexp, count int) string {
	b := make([]byte, 0, len(re.Sub)*count)
	for i := 0; i < count; i++ {
		for _, sub := range re.Sub {
			b = append(b, x.generateFromRegexp(sub)...)
		}
	}
	return string(b)
}

// Returns a non-negative pseudo-random number in [0,n).
// n must be > 0, but int31n does not check this; the caller must ensure it.
// randInt is simpler and faster than rand.Intn(n), because xeger just
// generates strings at random.
func randInt(n int) int {
	return int(src.Int63() % int64(n))
}

func isASCII(r rune) bool {
	return r >= 0 && r <= 127
}

func isInRanges(r rune, ranges []rune) bool {
	for i := 0; i < len(ranges); i += 2 {
		if r >= ranges[i] && r <= ranges[i+1] {
			return true
		}
	}
	return false
}
