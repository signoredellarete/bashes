# Direttive di sviluppo per Bashes

Queste istruzioni valgono per tutto il repository.

## Obiettivo del refactor

- Rifattorizzare Bashes come applicazione desktop cross-platform basata su Wails.
- Mantenere l'identita' del progetto: gestione rapida di server remoti, host, subsystem e sessioni operative.
- Evitare Electron per ora.
- Rendere l'app il piu' possibile autoconsistente: non dipendere da terminali, file manager, browser o client SSH gia' installati sull'OS per le funzionalita' principali.
- Integrare un terminale web-based, preferibilmente `xterm.js`, collegato a sessioni SSH gestite dal backend.
- Usare Go/Wails per il backend nativo, con gestione di sessioni SSH, streaming I/O, resize PTY, autenticazione e datastore locale.

## Dati e portabilita'

- Conservare la caratteristica dei dati salvati in JSON.
- Il file JSON deve restare facile da copiare/importare tra istanze diverse di Bashes.
- Non introdurre un database server.
- Se serve piu' robustezza, usare repository JSON con validazione, backup, locking e scrittura atomica.
- Preferire ID stabili univoci per host e subsystem, mantenendo compatibilita' o migrazione dai dati storici dove possibile.
- Tenere datastore, backup e file runtime dentro directory del progetto o in path configurabili, evitando scritture sparse nel sistema.

## Ambiente di sviluppo

- L'ambiente corrente puo' essere un server senza desktop grafico.
- Non chiedere all'utente di aprire browser o controllare manualmente la UI per verifiche ordinarie.
- Usare test headless, unit test, test di integrazione, build CLI, validazioni JSON e, quando disponibile, Playwright/headless browser.
- Se un test visuale richiede un server locale, scegliere porte libere dopo aver controllato i processi/servizi esistenti.
- Prima di avviare server, watcher o servizi persistenti, controllare possibili conflitti con processi gia' attivi.
- Non lasciare processi di sviluppo in esecuzione al termine del lavoro, salvo richiesta esplicita.

## Vincoli sul server

- Questo server ospita altre applicazioni, sia in container Docker sia direttamente sull'host.
- Non interferire con servizi, porte, container, volumi o file esterni al repository.
- Tenere file generati, cache, build e dati temporanei dentro questo repository o dentro `/tmp` quando appropriato.
- Non installare pacchetti di sistema se non strettamente necessario.
- Preferire toolchain locali al progetto, binari scaricati in directory del repo, o container temporanei solo dopo aver verificato che non creino conflitti.
- Se una dipendenza richiede rete o installazione, provare prima alternative gia' presenti; se e' davvero necessaria, usare il minimo indispensabile.

## Autonomia operativa

- Procedere in autonomia senza chiedere conferma per ogni passo ordinario.
- Fare assunzioni conservative e documentarle quando servono.
- Chiedere input solo se una scelta e' irreversibile, distruttiva, richiede credenziali, o puo' impattare servizi esterni al repo.
- Non eseguire comandi distruttivi su file non generati dal lavoro corrente senza richiesta esplicita.
- Non modificare file fuori da questo repository salvo necessita' esplicita e motivata.

## Git

- Fare commit e push a ogni modifica importante o milestone coerente.
- Non modificare l'autore Git originale del repository.
- Non cambiare `git config user.name` o `git config user.email` salvo richiesta esplicita.
- Usare l'autore Git gia' configurato nel repository/sessione.
- Prima di commit, controllare `git status` e includere solo file pertinenti.
- Non revertire modifiche non proprie senza richiesta esplicita.
- Scrivere messaggi di commit brevi, descrittivi e coerenti.
- Pushare su `origin` il branch corrente, salvo diversa indicazione dell'utente.

## Qualita' tecnica

- Mantenere l'app veloce e leggera come requisito primario.
- Evitare framework o astrazioni pesanti se non portano valore concreto.
- Separare chiaramente frontend, backend, datastore, sessioni SSH e packaging.
- Coprire con test il repository JSON, validazione input, migrazioni dati e gestione sessioni dove possibile.
- Non introdurre dipendenze runtime superflue.
- Preferire API strutturate a parsing ad hoc.
- Aggiornare la documentazione quando cambia il modo di installare, avviare, testare o migrare i dati.

## Verifiche minime prima di chiudere una modifica

- `git status --short`
- controlli sulle porte/processi se sono stati avviati server locali;
- test o build disponibili per la parte modificata;
- validazione JSON o test di migrazione quando vengono toccati i dati;
- riepilogo chiaro di cio' che e' stato cambiato e di cio' che non e' stato possibile verificare.

