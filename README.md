good
====

Take a closer look at your git history.

The goal of good is to analyze what you've been working on. I wrote good because I'm committing to git repositories that are both public and private, and I'd like to record what I'm working on.

Good takes an email and path as arguments and tries to find all of the commits it can:

	$ good --email=graham.abbott@gmail.com | sort -n -r -k3
           +js | 104000
           +py | 44554
          +css | 27418
         +html | 23340
           +go | 5343
          +txt | 3115
           +md | 2950
          +svg | 2006
          +erl | 1507
        +genie | 622
       +coffee | 386
         +atom | 333
         +json | 308
          +sql | 176
              ...
              
A commit history file is stored in your HOME directory so you can do some analysis of your own as well.