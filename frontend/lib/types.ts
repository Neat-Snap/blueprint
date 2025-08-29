export interface TokenResponse {
  token: string;
  success: boolean;
}

export interface SignupResponse {
  confirmation_id: string;
  success: boolean;
  message: string;
}
