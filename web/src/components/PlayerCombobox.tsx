import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronsUpDown } from "lucide-react";
import { listPlayers } from "@/api/players";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";

interface PlayerComboboxProps {
  value: string;
  onChange: (username: string) => void;
  exclude?: string[];
  placeholder?: string;
}

export default function PlayerCombobox({
  value,
  onChange,
  exclude = [],
  placeholder = "选择对手…",
}: PlayerComboboxProps) {
  const [open, setOpen] = useState(false);
  const { data: players = [] } = useQuery({
    queryKey: ["players"],
    queryFn: listPlayers,
    staleTime: 30_000,
  });

  const options = players.filter((p) => !exclude.includes(p.username));

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
          data-testid="player-combobox-trigger"
        >
          {value || placeholder}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
        <Command>
          <CommandInput placeholder="搜索玩家…" />
          <CommandList>
            <CommandEmpty>未找到玩家</CommandEmpty>
            <CommandGroup>
              {options.map((p) => (
                <CommandItem
                  key={p.id}
                  value={p.username}
                  onSelect={(selected) => {
                    onChange(selected);
                    setOpen(false);
                  }}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      value === p.username ? "opacity-100" : "opacity-0",
                    )}
                  />
                  {p.username}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
