export type UserRole = 'SUPER_ADMIN' | 'MANAGER';
export type UserStatus = 'PENDING_APPROVAL' | 'APPROVED' | 'REJECTED';

export interface UserDTO {
	id: string;
	full_name: string;
	email: string;
	role: UserRole;
	status: UserStatus;
	approved_by?: string;
	approved_at?: string;
	created_at: string;
}

export interface SignupRequest {
	full_name: string;
	email: string;
	password: string;
}

export interface LoginRequest {
	email: string;
	password: string;
}

export interface AuthTokensResponse {
	access_token: string;
	refresh_token: string;
	token_type: string;
	user: UserDTO;
}

export interface CreateMCPTokenRequest {
	label: string;
}

export interface MCPTokenResponse {
	token: string;
	label: string;
}
