# Decimal values

No fiscal value in this library ever passes through a `float64`. That is a
design decision, not an aesthetic preference.

## The problem with floating point

`float64` represents numbers in base 2. Values such as 0.1 and 0.07 have no
exact representation in that base, and the error accumulates:

```go
soma := 0.0
for range 100 {
    soma += 0.07
}
fmt.Println(soma) // 7.000000000000005
```

Seven reais becomes seven reais and an invisible fraction. When that value is
formatted to two decimal places the error disappears, but the moment it takes
part in a comparison, a subtraction or another multiplication, the cent shows
up — and SEFAZ rejects the invoice because a sum does not add up.

With [`tipos.Decimal`](https://pkg.go.dev/github.com/mschunke/gonfe/tipos#Decimal)
the same loop gives exactly `7.00`.

## How it works

`Decimal` holds an unscaled integer plus the number of decimal places. The value
`10.50` is the pair `(1050, 2)`. Every operation is done on integers, so there
is no representation error.

```go
a := tipos.D("10.50")       // 1050 with 2 places
b := tipos.D("0.255")       // 255 with 3 places

a.Somar(b)                  // 10.755  — takes the larger scale
a.Subtrair(b)               // 10.245
a.Multiplicar(b)            // 2.67750 — scales add up
a.MultiplicarCom(b, 2)      // 2.68    — rounded to 2 places
a.Percentual(tipos.D("18.00"), 2) // 1.89 — 18% of 10.50
```

## Building values

| Function | Use |
| --- | --- |
| `tipos.D("199.90")` | Literals in code; panics if the text is invalid |
| `tipos.ParseDecimal(s)` | External input, returns an error |
| `tipos.ParseDecimalBR(s)` | Brazilian format: `"1.234,56"` |
| `tipos.NovoDecimal(19990, 2)` | From cents, when your database stores integers |
| `tipos.DeInteiro(42)` | Integer with no decimal places |

The scale is preserved exactly as written: `tipos.D("10.50")` has two places,
`tipos.D("10.5")` has one, and the two are numerically equal for `Igual` and
`Comparar`.

!!! warning "Never convert from `float64`"

    There is no constructor taking a `float64`, and that is deliberate. If the
    value arrives as a `float64` from a system you do not control, format it at
    the precision you want before converting:

    ```go
    d, err := tipos.ParseDecimal(strconv.FormatFloat(v, 'f', 2, 64))
    ```

    `Float64()` exists for the opposite direction — display and approximate
    calculations — and must never feed a fiscal XML.

## Automatic scaling from the layout

Every field in the layout has a mandatory precision: `vProd` has two places,
`qCom` has four, `vUnCom` has ten. Writing `<vProd>25</vProd>` is a certain
rejection; the correct form is `<vProd>25.00</vProd>`.

You do not have to worry about this. The `nfe` package structures declare each
field's precision in a struct tag, and `NFe.Preparar` rescales everything before
serialising:

```go
type Prod struct {
    QCom   tipos.Decimal `xml:"qCom" dec:"4"`
    VUnCom tipos.Decimal `xml:"vUnCom" dec:"10"`
    VProd  tipos.Decimal `xml:"vProd" dec:"2"`
}
```

```go
prod.QCom = tipos.D("10")   // this is what you write
// after Preparar, the XML comes out as <qCom>10.0000</qCom>
```

## Rounding

Rounding is **commercial**: half away from zero, which is the rule the SEFAZ
manuals adopt.

```go
tipos.D("2.345").ComCasas(2)  // 2.35
tipos.D("2.344").ComCasas(2)  // 2.34
tipos.D("-2.345").ComCasas(2) // -2.35
tipos.D("10.5").ComCasas(0)   // 11
```

That differs from banker's rounding (half to even), used by several languages,
and from truncation. If your ERP rounds differently, the totals may diverge by a
cent — the library's validation tolerates that difference per item, as the
layout itself does.

## Round per item, sum afterwards

The order of operations matters and is fixed by the layout: tax is computed and
rounded **per item**, and the total is the sum of the already-rounded values.

```go
// Right: round each item, then add.
item1 := tipos.D("25.00").Percentual(tipos.D("1.65"), 2)  // 0.41
item2 := tipos.D("74.70").Percentual(tipos.D("1.65"), 2)  // 1.23
total := item1.Somar(item2)                               // 1.64

// Wrong: add the bases and round at the end.
tipos.D("99.70").Percentual(tipos.D("1.65"), 2)           // 1.65
```

One cent apart. `CalcularTotais` does it the right way.

## Serialisation

`Decimal` implements both the XML and the JSON interfaces. In JSON the value
comes out as a string, so that precision is not lost passing through a parser
that uses `float64`:

```json
{"vProd": "1250.00", "qCom": "10.0000"}
```

When reading, both a JSON string and a JSON number are accepted.
