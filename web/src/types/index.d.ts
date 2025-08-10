export interface Gate {
  id: string;
  live_url: string;
  shadow_url: string;
  created_at: string;
}

export interface Request {
  id: string;
  method: string;
  path: string;
  headers: Record<string, string>;
  body: string;
  created_at: string;

  responses: Response[];
  diff: Diff;
}

export interface Response {
  id: string;
  status_code: number;
  headers: Record<string, string>;
  body: string;
  created_at: string;
}

export interface Diff {
  id: string;
  content: string;
}
