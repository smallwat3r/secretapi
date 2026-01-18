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

export interface ConfigResponse {
  max_secret_size: number;
  expiry_options: string[];
}

export type Expiry = string;
