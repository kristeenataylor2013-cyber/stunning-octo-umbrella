import argparse
import logging

from .config import Config


def main() -> None:
    ap = argparse.ArgumentParser(prog="client_docs_agent")
    ap.add_argument("command", choices=["run", "login-interactive"])
    args = ap.parse_args()
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    from . import agent
    cfg = Config.from_env()
    try:
        if args.command == "run":
            agent.run(cfg)
        else:
            agent.login_interactive(cfg)
    except agent.IncompleteRun as exc:
        raise SystemExit(str(exc)) from None


if __name__ == "__main__":
    main()
