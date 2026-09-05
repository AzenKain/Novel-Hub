export interface KoboSetup {
  endpoint_url: string;
  /** True when URL points to loopback which Kobo cannot resolve. */
  is_local_address: boolean;
}
