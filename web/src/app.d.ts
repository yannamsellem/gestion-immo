import type { UserDTO } from '#lib/modules/auth/types';
import type { BackendApi } from '#lib/server/backend-api';

declare global {
	namespace App {
		interface Platform {
			env: Env & {
				API_BASE_URL?: string;
				PUBLIC_API_BASE_URL?: string;
				CF_ACCESS_CLIENT_ID?: string;
				CF_ACCESS_CLIENT_SECRET?: string;
			};
			ctx: ExecutionContext;
			caches: CacheStorage;
			cf?: IncomingRequestCfProperties;
		}

		interface Locals {
			user: UserDTO | null;
			selectedEntityId: 'ALL' | string;
			selectedPeriodMonth: string;
			api: BackendApi;
		}
	}
}

export {};
