piper := 'TTY=0 piper -p piper.cue' + if env("DEBUG", "0") == '1' { ' --log-level=debug' } else { '' }

ship:
    {{ piper }} do ship
