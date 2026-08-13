-- Knihovna pro snímání uživatelského inputu je Bufio

    reader := bufio.NewReader(os.Stdin)
    input, _ := reader.ReadString('\n')
