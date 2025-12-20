export interface ApiErrorResponse {
  error?: string;
  remaining_attempts?: number;
}

export interface CreateResponse {
  read_url: string;
  passcode: string;
  expires_at: string;
}

export interface ReadResponse {
  secret: string;
}

export type Expiry = '1h' | '6h' | '1d' | '3d';
