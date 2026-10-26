// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
import type { UserDTO } from '$lib/api/client';

declare global {
	namespace App {
		interface Error {
			code?: string;
		}
		interface Locals {
			accessToken?: string;
			user?: UserDTO | null;
		}
		interface PageData {
			user?: UserDTO | null;
			activeEntityId?: string;
		}
		interface Platform {
			env?: {
				PUBLIC_API_BASE_URL?: string;
			};
		}
	}
}

export {};
