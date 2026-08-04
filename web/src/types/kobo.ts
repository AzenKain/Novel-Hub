export interface KoboSetup {
  endpoint_url: string;
  /** True when the URL points at loopback — a Kobo cannot resolve it, so the UI warns. */
  is_local_address: boolean;
}
