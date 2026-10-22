import os
from dataclasses import dataclass
from pathlib import Path


def _load_dotenv(path: str = ".env") -> None:
    p = Path(path)
    if not p.exists():
        return
    for line in p.read_text().splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            k, v = line.split("=", 1)
            os.environ.setdefault(k.strip(), v.strip())


def _secret(name: str) -> str | None:
    val = os.environ.get(name)
    if val:
        return val
    try:
        import keyring
        return keyring.get_password("client-docs-agent", name)
    except Exception:
        return None


@dataclass
class Config:
    base_url: str
    login_path: str
    clients_path: str
    username: str | None
    password: str | None
    output_dir: Path
    session_state: Path
    sel_username: str
    sel_password: str
    sel_submit: str
    sel_client_link: str
    sel_next_page: str
    sel_doc_link: str
    delay: float = 1.0

    @classmethod
    def from_env(cls) -> "Config":
        _load_dotenv()
        e = os.environ
        if not e.get("SITE_BASE_URL") or not e.get("OUTPUT_DIR"):
            raise SystemExit("SITE_BASE_URL and OUTPUT_DIR are required")
        out = Path(e["OUTPUT_DIR"]).expanduser()
        return cls(
            base_url=e["SITE_BASE_URL"].rstrip("/"),
            login_path=e.get("SITE_LOGIN_PATH", "/login"),
            clients_path=e.get("SITE_CLIENTS_PATH", "/clients"),
            username=_secret("SITE_USERNAME"),
            password=_secret("SITE_PASSWORD"),
            output_dir=out,
            session_state=Path(e.get("SESSION_STATE", str(out.parent / "session.json"))).expanduser(),
            sel_username=e.get("SEL_USERNAME", "input[name=username]"),
            sel_password=e.get("SEL_PASSWORD", "input[name=password]"),
            sel_submit=e.get("SEL_SUBMIT", "button[type=submit]"),
            sel_client_link=e.get("SEL_CLIENT_LINK", "a.client-link"),
            sel_next_page=e.get("SEL_NEXT_PAGE", "a.next"),
            sel_doc_link=e.get("SEL_DOC_LINK", "a.document-link"),
            delay=float(e.get("REQUEST_DELAY", "1.0")),
        )
