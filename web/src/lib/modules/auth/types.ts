import type { components } from '#lib/core/api/openapi';

export type UserRole = 'SUPER_ADMIN' | 'MANAGER' | string;
export type UserStatus = 'PENDING_APPROVAL' | 'APPROVED' | 'REJECTED' | string;

export type UserDTO = components['schemas']['UserDTO'];
export type SignupRequest = components['schemas']['SignupRequest'];
export type LoginRequest = components['schemas']['LoginRequest'];
export type AuthTokensResponse = components['schemas']['AuthTokensResponse'];
export type CreateMCPTokenRequest = components['schemas']['CreateMCPTokenRequest'];
export type MCPTokenResponse = components['schemas']['MCPTokenResponse'];
