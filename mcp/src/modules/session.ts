export let current_user: string;

export async function middleware(key: string) {
    current_user = key;
    return true;
}
