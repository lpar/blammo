
# Blammo

This is a simple console logger for Go which supports:

 - High performance logging (zero allocations)
 - An API modeled on zerolog
 - Human readable output, not JSON
 - Logging errors to stderr and everything else to stdout
 - Optional logging to files
 - Not much code

I liked zerolog's API and speed, but its console logging was slow, it lacked
stdout/stderr separation, and it was a lot of code -- mostly because it
implemented lots of functionality I didn't need.

If you want to log to systemd, JSON APIs or binary files, use zerolog. If you
just want logging for humans at high speed with minimal code, this is a smaller
and faster option.

Simple usage:

    import "github.com/lpar/blammo/log"

    // turn on debug output for default logger at runtime
    log.SetDebug(true)

    log.Info().Msg("Hello sailor")
    log.Debug().Int("x", 6).Msg("Debug info")

Hopefully you'll agree that it's better than bad, it's good.

## Design notes

I believe in minimizing the number of logging levels. Over many years, these
are the ones I've found I use:

 - *Info*: Normal information, progress messages, job output.
 - *Error*: Something has definitely gone wrong.
 - *Warning*: Something has happened which may be a program error or may be a 
   human error, and could be worth investigating.
 - *Debug*: Trace information to be turned on when debugging.

An example of a situation where a warning is appropriate might be an invalid
digital signature on a JSON Web Token. It *could* indicate an error in the
decoding, but it could also be caused by someone trying to bypass security
using a JWT with a fake signature.

There's no Fatal level in my world, because it's almost never appropriate to
crash out in an uncontrolled fashion after detecting an error. If you're in one
of the situations where it *is* appropriate, just do something like:

    log.Error().Msg("fatal error")
    log.Close()
    os.Exit(1)

Errors and warnings are sent to a separate stream by default because that's the
Unix convention since time immemorial. Also, on the cloud hosting I use, output
to stderr gets highlighted to distinguish it from normal logging output. I'm aware
that you can't reliably reconstruct the message sequence from stderr and stdout when
they're separated like this; that's why we have timestamps on every line, right?

I haven't implemented the whole zerolog API. For example, if you want to dump
out a slice or array of anything other than a few bytes, you'll have to write a
loop,as there's no equivalent of zerolog's Array type. I think this makes sense
from the point of view of line length and simplicity. I also haven't
implemented special message appenders for types which are Stringers, such as IP
addresses; just use their `.String()` method.

An added option zerolog lacks is the `Msgf()` method. This works like
`fmt.Printf`, and is consequently relatively slow, but is there to make it easy
to migrate code from other loggers that use a Printf-style interface (like my
own [first cut at a logging wrapper](http://github.com/lpar/log). It's also
convenient for user-visible error messages where speed isn't a major concern.

