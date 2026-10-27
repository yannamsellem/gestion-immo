import type { HttpClient } from '#lib/core/api/http-client';
import type {
	AuthTokensResponse,
	CreateMCPTokenRequest,
	LoginRequest,
	MCPTokenResponse,
	SignupRequest,
	UserDTO
} from './types';

export class AuthApiService {
	constructor(private readonly http: HttpClient) {}

	signup(payload: SignupRequest): Promise<UserDTO> {
		return this.http.requestJson<UserDTO>('/api/v1/auth/signup', {
			method: 'POST',
			body: JSON.stringify(payload)
		});
	}

	login(payload: LoginRequest): Promise<AuthTokensResponse> {
		return this.http.requestJson<AuthTokensResponse>('/api/v1/auth/login', {
			method: 'POST',
			body: JSON.stringify(payload)
		});
	}

	refresh(refreshToken: string): Promise<AuthTokensResponse> {
		return this.http.requestJson<AuthTokensResponse>('/api/v1/auth/refresh', {
			method: 'POST',
			body: JSON.stringify({ refresh_token: refreshToken })
		});
	}

	logout(refreshToken: string): Promise<void> {
		return this.http.requestJson<void>('/api/v1/auth/logout', {
			method: 'POST',
			body: JSON.stringify({ refresh_token: refreshToken })
		});
	}

	createMcpToken(payload: CreateMCPTokenRequest): Promise<MCPTokenResponse> {
		return this.http.requestJson<MCPTokenResponse>('/api/v1/auth/mcp-tokens', {
			method: 'POST',
			body: JSON.stringify(payload)
		});
	}

	listPendingUsers(): Promise<UserDTO[]> {
		return this.http.requestJson<UserDTO[]>('/api/v1/admin/users/pending');
	}

	approveUser(userId: string): Promise<UserDTO> {
		return this.http.requestJson<UserDTO>(
			`/api/v1/admin/users/${encodeURIComponent(userId)}/approve`,
			{
				method: 'POST'
			}
		);
	}

	rejectUser(userId: string): Promise<UserDTO> {
		return this.http.requestJson<UserDTO>(
			`/api/v1/admin/users/${encodeURIComponent(userId)}/reject`,
			{
				method: 'POST'
			}
		);
	}
}
