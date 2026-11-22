// Stand-in for `xmcp/headers`: the headers of the request being served, with
// the bearer a test sets through setAuthorization.

let authorization = "Bearer tiu_pat_test";

export function setAuthorization(value) {
    authorization = value;
}

export function headers() {
    return { authorization };
}
