package sprig

import (
	"errors"
	"math/rand"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	ttemplate "text/template"
	"time"

	util "github.com/Masterminds/goutils"
	"github.com/huandu/xstrings"
	"github.com/shopspring/decimal"
)

// TxtFuncMap returns a 'text/template'.FuncMap
func TxtFuncMap() ttemplate.FuncMap {
	return ttemplate.FuncMap(GenericFuncMap())
}

// GenericFuncMap returns a copy of the basic function map as a map[string]interface{}.
func GenericFuncMap() map[string]interface{} {
	gfm := make(map[string]interface{}, len(genericMap))
	for k, v := range genericMap {
		gfm[k] = v
	}
	return gfm
}

var genericMap = map[string]interface{}{
	"hello": func() string { return "Hello!" },

	// Date functions
	"ago":              dateAgo,
	"date":             date,
	"date_in_zone":     dateInZone,
	"date_modify":      dateModify,
	"dateInZone":       dateInZone,
	"dateModify":       dateModify,
	"duration":         duration,
	"durationRound":    durationRound,
	"htmlDate":         htmlDate,
	"htmlDateInZone":   htmlDateInZone,
	"must_date_modify": mustDateModify,
	"mustDateModify":   mustDateModify,
	"mustToDate":       mustToDate,
	"now":              time.Now,
	"toDate":           toDate,
	"unixEpoch":        unixEpoch,

	// Strings
	"abbrev":       abbrev,
	"abbrevboth":   abbrevboth,
	"trunc":        trunc,
	"trim":         strings.TrimSpace,
	"upper":        strings.ToUpper,
	"lower":        strings.ToLower,
	"title":        strings.Title,
	"untitle":      untitle,
	"substr":       substring,
	"repeat":       func(count int, str string) string { return strings.Repeat(str, count) },
	"trimall":      func(a, b string) string { return strings.Trim(b, a) },
	"trimAll":      func(a, b string) string { return strings.Trim(b, a) },
	"trimSuffix":   func(a, b string) string { return strings.TrimSuffix(b, a) },
	"trimPrefix":   func(a, b string) string { return strings.TrimPrefix(b, a) },
	"nospace":      util.DeleteWhiteSpace,
	"initials":     initials,
	"randAlphaNum": randAlphaNumeric,
	"randAlpha":    randAlpha,
	"randAscii":    randAscii,
	"randNumeric":  randNumeric,
	"swapcase":     util.SwapCase,
	"shuffle":      xstrings.Shuffle,
	"snakecase":    xstrings.ToSnakeCase,
	"camelcase":    xstrings.ToPascalCase,
	"kebabcase":    xstrings.ToKebabCase,
	"wrap":         func(l int, s string) string { return util.Wrap(s, l) },
	"wrapWith":     func(l int, sep, str string) string { return util.WrapCustom(str, l, sep, true) },
	"contains":     func(substr string, str string) bool { return strings.Contains(str, substr) },
	"hasPrefix":    func(substr string, str string) bool { return strings.HasPrefix(str, substr) },
	"hasSuffix":    func(substr string, str string) bool { return strings.HasSuffix(str, substr) },
	"quote":        quote,
	"squote":       squote,
	"cat":          cat,
	"indent":       indent,
	"nindent":      nindent,
	"replace":      replace,
	"plural":       plural,
	"sha1sum":      sha1sum,
	"sha256sum":    sha256sum,
	"sha512sum":    sha512sum,
	"adler32sum":   adler32sum,
	"toString":     strval,

	"atoi":      func(a string) int { i, _ := strconv.Atoi(a); return i },
	"int64":     toInt64,
	"int":       toInt,
	"float64":   toFloat64,
	"seq":       seq,
	"toDecimal": toDecimal,

	"split":     split,
	"splitList": func(sep, orig string) []string { return strings.Split(orig, sep) },
	"splitn":    splitn,
	"toStrings": strslice,

	"until":     until,
	"untilStep": untilStep,

	"add1": func(i interface{}) int64 { return toInt64(i) + 1 },
	"add": func(i ...interface{}) int64 {
		var a int64 = 0
		for _, b := range i {
			a += toInt64(b)
		}
		return a
	},
	"sub": func(a, b interface{}) int64 { return toInt64(a) - toInt64(b) },
	"div": func(a, b interface{}) int64 { return toInt64(a) / toInt64(b) },
	"mod": func(a, b interface{}) int64 { return toInt64(a) % toInt64(b) },
	"mul": func(a interface{}, v ...interface{}) int64 {
		val := toInt64(a)
		for _, b := range v {
			val = val * toInt64(b)
		}
		return val
	},
	"randInt": func(min, max int) int { return rand.Intn(max-min) + min },
	"add1f": func(i interface{}) float64 {
		return execDecimalOp(i, []interface{}{1}, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Add(d2) })
	},
	"addf": func(i ...interface{}) float64 {
		a := interface{}(float64(0))
		return execDecimalOp(a, i, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Add(d2) })
	},
	"subf": func(a interface{}, v ...interface{}) float64 {
		return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Sub(d2) })
	},
	"divf": func(a interface{}, v ...interface{}) float64 {
		return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Div(d2) })
	},
	"mulf": func(a interface{}, v ...interface{}) float64 {
		return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Mul(d2) })
	},
	"biggest": max,
	"max":     max,
	"min":     min,
	"maxf":    maxf,
	"minf":    minf,
	"ceil":    ceil,
	"floor":   floor,
	"round":   round,

	"join":      join,
	"sortAlpha": sortAlpha,

	// Defaults
	"default":          dfault,
	"empty":            empty,
	"coalesce":         coalesce,
	"all":              all,
	"any":              any,
	"compact":          compact,
	"mustCompact":      mustCompact,
	"fromJson":         fromJson,
	"toJson":           toJson,
	"toPrettyJson":     toPrettyJson,
	"toRawJson":        toRawJson,
	"mustFromJson":     mustFromJson,
	"mustToJson":       mustToJson,
	"mustToPrettyJson": mustToPrettyJson,
	"mustToRawJson":    mustToRawJson,
	"ternary":          ternary,
	"deepCopy":         deepCopy,
	"mustDeepCopy":     mustDeepCopy,

	// Reflection
	"typeOf":     typeOf,
	"typeIs":     typeIs,
	"typeIsLike": typeIsLike,
	"kindOf":     kindOf,
	"kindIs":     kindIs,
	"deepEqual":  reflect.DeepEqual,

	// Paths:
	"base":  path.Base,
	"dir":   path.Dir,
	"clean": path.Clean,
	"ext":   path.Ext,
	"isAbs": path.IsAbs,

	// Filepaths:
	"osBase":  filepath.Base,
	"osClean": filepath.Clean,
	"osDir":   filepath.Dir,
	"osExt":   filepath.Ext,
	"osIsAbs": filepath.IsAbs,

	// Encoding:
	"b64enc": base64encode,
	"b64dec": base64decode,
	"b32enc": base32encode,
	"b32dec": base32decode,

	// Data Structures:
	"tuple":              list,
	"list":               list,
	"dict":               dict,
	"get":                get,
	"set":                set,
	"unset":              unset,
	"hasKey":             hasKey,
	"pluck":              pluck,
	"keys":               keys,
	"pick":               pick,
	"omit":               omit,
	"merge":              merge,
	"mergeOverwrite":     mergeOverwrite,
	"mustMerge":          mustMerge,
	"mustMergeOverwrite": mustMergeOverwrite,
	"values":             values,

	"append": push, "push": push,
	"mustAppend": mustPush, "mustPush": mustPush,
	"prepend":     prepend,
	"mustPrepend": mustPrepend,
	"first":       first,
	"mustFirst":   mustFirst,
	"rest":        rest,
	"mustRest":    mustRest,
	"last":        last,
	"mustLast":    mustLast,
	"initial":     initial,
	"mustInitial": mustInitial,
	"reverse":     reverse,
	"mustReverse": mustReverse,
	"uniq":        uniq,
	"mustUniq":    mustUniq,
	"without":     without,
	"mustWithout": mustWithout,
	"has":         has,
	"mustHas":     mustHas,
	"slice":       slice,
	"mustSlice":   mustSlice,
	"concat":      concat,
	"dig":         dig,
	"chunk":       chunk,
	"mustChunk":   mustChunk,

	// Crypto:
	"bcrypt":                   bcrypt,
	"htpasswd":                 htpasswd,
	"genPrivateKey":            generatePrivateKey,
	"derivePassword":           derivePassword,
	"buildCustomCert":          buildCustomCertificate,
	"genCA":                    generateCertificateAuthority,
	"genCAWithKey":             generateCertificateAuthorityWithPEMKey,
	"genSelfSignedCert":        generateSelfSignedCertificate,
	"genSelfSignedCertWithKey": generateSelfSignedCertificateWithPEMKey,
	"genSignedCert":            generateSignedCertificate,
	"genSignedCertWithKey":     generateSignedCertificateWithPEMKey,
	"encryptAES":               encryptAES,
	"decryptAES":               decryptAES,
	"randBytes":                randBytes,

	// UUIDs:
	"uuidv4": uuidv4,

	// SemVer:
	"semver":        semver,
	"semverCompare": semverCompare,

	// Flow Control:
	"fail": func(msg string) (string, error) { return "", errors.New(msg) },

	// Regex
	"regexMatch":                 regexMatch,
	"mustRegexMatch":             mustRegexMatch,
	"regexFindAll":               regexFindAll,
	"mustRegexFindAll":           mustRegexFindAll,
	"regexFind":                  regexFind,
	"mustRegexFind":              mustRegexFind,
	"regexReplaceAll":            regexReplaceAll,
	"mustRegexReplaceAll":        mustRegexReplaceAll,
	"regexReplaceAllLiteral":     regexReplaceAllLiteral,
	"mustRegexReplaceAllLiteral": mustRegexReplaceAllLiteral,
	"regexSplit":                 regexSplit,
	"mustRegexSplit":             mustRegexSplit,
	"regexQuoteMeta":             regexQuoteMeta,

	// URLs:
	"urlParse": urlParse,
	"urlJoin":  urlJoin,
}
