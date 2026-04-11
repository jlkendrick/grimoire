import rich 

from rich.console import Console
from rich.panel import Panel


def print_rich_panel(title: str):
	console = Console()

	console.print(
		Panel.fit(
			f"[bold green]Hello from [cyan]Rich[/cyan]! :sparkles:\nThis is a beautiful message rendered with the rich library.",
			title=f"[bold magenta]{title}!",
			border_style="bright_blue"
		)
	)