export interface RingInstance {
  id: string;
  state: string;
  address: string;
  timestamp: string;
  registered_timestamp: string;
  read_only: boolean;
  read_only_updated_timestamp: string;
  zone: string;
  tokens: number[];
}

export interface RingResponse {
  shards: RingInstance[];
  now: string;
}
