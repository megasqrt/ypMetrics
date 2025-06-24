package helper


func AssignIfNotEmpty[T comparable](dst *T, src T) {
    var zero T
    if src != zero {
        *dst = src
    }
}