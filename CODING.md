# Go coding style rules adopted

1. Use `x := []X_TYPE{}` when initializing an empty array - dont use `x := make(X_TYPE, 0)`
   Think twice whenever you use `x := make(X_TYPE, n)` for `n>0`
   Note that you just added n empty items to the array.
   Think 10 times if you ever use/see `x := make(X_TYPE, n)` followed by `x = append (x, something)`
