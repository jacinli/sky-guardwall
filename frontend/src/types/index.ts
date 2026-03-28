export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface IptablesRule {
  id: number
  line_type: 'policy' | 'chain' | 'rule'
  chain: string
  target: string
  protocol: string
  source: string
  dest: string
  src_port: string
  dst_port: string
  in_iface: string
  out_iface: string
  extra: string
  raw_line: string
  synced_at: string
  created_at: string
}

export interface IptablesRulesResponse {
  rules: IptablesRule[]
  total: number
  chains: string[]
  last_synced_at?: string
  sync_has_error?: boolean
  sync_error_msg?: string
}

export interface ManagedRule {
  id: number
  description: string
  chain: string
  src_ip: string
  protocol: string
  dst_port: number
  target: string
  iptables_args: string
  is_applied: boolean
  created_at: string
  updated_at: string
}

export interface AddRuleRequest {
  description?: string
  chain: string
  src_ip?: string
  protocol: string
  dst_port?: number
  target: string
}
