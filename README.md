# dicelint

Dice notation like `3d6+2` or `2d20kh1` shows up everywhere in tabletop game
tooling: character sheets, loot tables, VTT macros, combat simulators. It's
just text, so it drifts. Someone pastes `0d6` into a loot table by mistake,
or writes `2d20kh5` meaning "keep the best 5" when only 2 dice were rolled,
and nothing catches it until the numbers look wrong at the table.

dicelint parses dice expressions and reports problems the way a compiler
reports problems: a line, a column, and a caret under the exact token.

## Usage

```
go build -o dicelint .
./dicelint rolls.txt
```

Given a file `rolls.txt`:

```
3d6+2
0d6
2d20kh5
1d1
(3d6
```

dicelint prints:

```
rolls.txt:2:1: error: dice count is 0, this expression always rolls nothing [zero-dice-count]
    0d6
    ^
rolls.txt:3:1: error: modifier affects 5 dice but only 2 are rolled [keep-drop-exceeds-count]
    2d20kh5
    ^
rolls.txt:4:1: warning: a d1 always rolls 1, consider replacing it with the constant 1 [one-sided-die]
    1d1
    ^
rolls.txt:5:5: error: expected ')' to close '(' opened at line 5, column 1 [syntax]
    (3d6
        ^
```

Exit status is 1 if any finding is an error, 0 if the file only produced
warnings (or nothing at all).

## Notation supported

- Arithmetic: `+ - * /` and parentheses
- Dice terms: `NdM`, with `N` optional (defaults to 1, so `d20` is valid)
- Percentile dice: `d%`
- Keep/drop modifiers: `kh`, `kl`, `dh`, `dl`, each optionally followed by a
  count, e.g. `4d6kh3`, `2d20dl1`

## Checks implemented so far

| Rule | Meaning |
|---|---|
| `zero-dice-count` | `0d6` rolls nothing |
| `huge-dice-count` | dice counts over 1000, usually a typo |
| `zero-sided-die` | `d0` is not a die |
| `one-sided-die` | `d1` always rolls 1 |
| `modifier-zero-count` | `kh0` / `dl0` keep or drop nothing |
| `keep-drop-exceeds-count` | `2d6kh5` asks to keep more dice than were rolled |
| `syntax` | malformed expressions, reported at the offending token |

More checks are planned; the rule set above is a starting point, not the
final list.

## Status

Early skeleton. The lexer, parser, and lint rules above work end to end on
real input, but there's no test suite yet and the rule set is deliberately
small.

## License

MIT, see [LICENSE](LICENSE).
