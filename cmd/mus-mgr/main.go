package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/semver"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/ikafly144/au_mod_installer/server/model"
	gormrepo "github.com/ikafly144/au_mod_installer/server/repository/gorm"
)

type parsedFile struct {
	Path           string
	URLs           []string
	Type           string
	ExtractPath    *string
	TargetPlatform string
}

func parseFileFlag(val string) *parsedFile {
	pf := &parsedFile{
		Type:           string(model.FileTypeArchive),
		TargetPlatform: string(model.TargetPlatformAny),
	}

	if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
		if !strings.Contains(val, "type=") && !strings.Contains(val, "path=") && !strings.Contains(val, "url=") {
			pf.URLs = append(pf.URLs, val)
			return pf
		}
	}

	if !strings.Contains(val, "=") {
		pf.Path = val
		return pf
	}

	for _, part := range strings.Split(val, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			kStr := strings.TrimSpace(kv[0])
			vStr := strings.TrimSpace(kv[1])
			switch kStr {
			case "path":
				pf.Path = vStr
			case "type":
				pf.Type = vStr
			case "url":
				pf.URLs = append(pf.URLs, strings.Split(vStr, "|")...)
			case "extract_path":
				pf.ExtractPath = &vStr
			case "target_platform":
				pf.TargetPlatform = vStr
			}
		}
	}

	if pf.Path == "" && len(pf.URLs) == 0 {
		pf.Path = val
	}

	return pf
}

func main() {
	var dbUrl string

	app := &cli.App{
		Name:  "mus-mgr",
		Usage: "Manage the au_mod_installer server database",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "db",
				Usage:       "PostgreSQL connection string",
				EnvVars:     []string{"DATABASE_URL"},
				Destination: &dbUrl,
				Required:    true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "migrate",
				Usage: "Migrate the database schema",
				Action: func(c *cli.Context) error {
					db, err := gorm.Open(postgres.Open(dbUrl))
					if err != nil {
						return err
					}
					repo := gormrepo.NewGormRepository(db)
					if err := repo.Migrate(); err != nil {
						return err
					}
					fmt.Println("Migration successful.")
					return nil
				},
			},
			{
				Name:  "mod",
				Usage: "Manage mods",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add a new mod",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "id", Usage: "Mod ID (default: uuid)", Value: ""},
							&cli.StringFlag{Name: "name", Required: true},
							&cli.StringFlag{Name: "author", Required: true},
							&cli.StringFlag{Name: "desc", Required: true},
						},
						Action: func(c *cli.Context) error {
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							id := c.String("id")
							if id == "" {
								id = uuid.New().String()
							}

							mod := &model.ModDetails{
								ID:          id,
								Name:        c.String("name"),
								Author:      c.String("author"),
								Description: c.String("desc"),
							}

							if _, err := repo.CreateMod(mod); err != nil {
								return err
							}
							fmt.Printf("Created mod: %s\n", mod.ID)
							return nil
						},
					},
					{
						Name:  "list",
						Usage: "List mods",
						Action: func(c *cli.Context) error {
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							ids, _, err := repo.GetModIds("", 100)
							if err != nil {
								return err
							}
							for _, id := range ids {
								mod, err := repo.GetModDetails(id)
								if err != nil {
									continue
								}
								fmt.Printf("%s\t%s\t%s\n", mod.ID, mod.Name, mod.Author)
							}
							return nil
						},
					},

					{
						Name:      "info",
						Usage:     "Get details of a mod",
						ArgsUsage: "<mod-id>",
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								return fmt.Errorf("mod-id required")
							}
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							mod, err := repo.GetModDetails(c.Args().First())
							if err != nil {
								return err
							}
							b, err := json.MarshalIndent(mod, "", "  ")
							if err != nil {
								return err
							}
							fmt.Println(string(b))
							return nil
						},
					},
					{
						Name:      "delete",
						Usage:     "Delete a mod",
						ArgsUsage: "<mod-id>",
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								return fmt.Errorf("mod-id required")
							}
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							if err := repo.DeleteMod(c.Args().First()); err != nil {
								return err
							}
							fmt.Println("Deleted mod:", c.Args().First())
							return nil
						},
					},
				},
			},
			{
				Name:  "version",
				Usage: "Manage mod versions",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add a new version for a mod",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "mod-id", Required: true, Usage: "Target mod ID"},
							&cli.StringFlag{Name: "version-id", Usage: "Version ID (default: auto-incremented SemVer)"},
							&cli.StringSliceFlag{Name: "file", Usage: "Files to add. Multiple flags supported. Format: path=...,type=...,url=...,extract_path=...,target_platform=... or direct URL/Path"},
							&cli.StringSliceFlag{Name: "dependency", Usage: "Dependencies to add. Multiple flags supported. Format: mod_id:version_id:type (type is optional, default: required)"},
							&cli.BoolFlag{Name: "set-latest", Usage: "Set this version as the latest version for the mod"},
						},
						Action: func(c *cli.Context) error {
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							modId := c.String("mod-id")
							verId := c.String("version-id")
							if verId == "" {
								existingIds, err := repo.GetModVersionIds(modId)
								if err != nil {
									return fmt.Errorf("failed to fetch existing versions for auto-increment: %w", err)
								}
								highest := ""
								for _, id := range existingIds {
									semId := id
									if !strings.HasPrefix(semId, "v") {
										semId = "v" + semId
									}
									if semver.IsValid(semId) {
										if highest == "" || semver.Compare(semId, highest) > 0 {
											highest = semId
										}
									}
								}
								if highest == "" {
									verId = "1.0.0"
								} else {
									base := strings.SplitN(strings.TrimPrefix(highest, "v"), "-", 2)[0]
									base = strings.SplitN(base, "+", 2)[0]
									parts := strings.Split(base, ".")
									var major, minor, patch int
									if len(parts) > 0 {
										major, _ = strconv.Atoi(parts[0])
									}
									if len(parts) > 1 {
										minor, _ = strconv.Atoi(parts[1])
									}
									if len(parts) > 2 {
										patch, _ = strconv.Atoi(parts[2])
									}
									verId = fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
								}
								fmt.Printf("Auto-generated version ID: %s\n", verId)
							}

							ver := &model.ModVersionDetails{
								ID: verId,
							}
							var deps []model.ModVersionDependency
							for _, d := range c.StringSlice("dependency") {
								parts := strings.Split(d, ":")
								if len(parts) >= 2 {
									depType := model.DependencyTypeRequired
									if len(parts) > 2 {
										depType = model.DependencyType(parts[2])
									}
									deps = append(deps, model.ModVersionDependency{
										ModID:          parts[0],
										VersionID:      parts[1],
										DependencyType: depType,
									})
								}
							}
							ver.Dependencies = deps

							fileFlags := c.StringSlice("file")
							for _, fileFlag := range fileFlags {
								pf := parseFileFlag(fileFlag)

								var filename string
								var size int64
								hasher := sha256.New()

								if pf.Path != "" {
									filename = filepath.Base(pf.Path)
									file, err := os.Open(pf.Path)
									if err != nil {
										return fmt.Errorf("failed to open file %s: %w", pf.Path, err)
									}

									stat, err := file.Stat()
									if err != nil {
										file.Close()
										return fmt.Errorf("failed to stat file %s: %w", pf.Path, err)
									}
									size = stat.Size()

									if _, err := io.Copy(hasher, file); err != nil {
										file.Close()
										return fmt.Errorf("failed to hash file: %w", err)
									}
									file.Close()
								} else if len(pf.URLs) > 0 {
									dlURL := pf.URLs[0]
									parsedPath := dlURL
									if strings.Contains(dlURL, "?") {
										parsedPath = strings.SplitN(dlURL, "?", 2)[0]
									}
									filename = filepath.Base(parsedPath)
									if filename == "" || filename == "/" || filename == "." {
										filename = "downloaded_file"
									}

									fmt.Printf("Downloading %s to compute size and hash...\n", dlURL)
									resp, err := http.Get(dlURL)
									if err != nil {
										return fmt.Errorf("failed to download url %s: %w", dlURL, err)
									}
									if resp.StatusCode != 200 {
										resp.Body.Close()
										return fmt.Errorf("failed to download url %s: status %d", dlURL, resp.StatusCode)
									}

									size, err = io.Copy(hasher, resp.Body)
									resp.Body.Close()
									if err != nil {
										return fmt.Errorf("failed to read from url %s: %w", dlURL, err)
									}
								} else {
									return fmt.Errorf("invalid file specifier: neither path nor url provided")
								}

								hashStr := hex.EncodeToString(hasher.Sum(nil))

								verFile := model.ModVersionFile{
									ID:             uuid.New().String(),
									Filename:       filename,
									ContentType:    model.FileType(pf.Type),
									Size:           size,
									ExtractPath:    pf.ExtractPath,
									TargetPlatform: model.TargetPlatform(pf.TargetPlatform),
									Hashes:         map[string]string{"sha256": hashStr},
									Downloads:      pf.URLs,
								}
								ver.Files = append(ver.Files, verFile)
							}

							if _, err := repo.CreateModVersion(modId, ver); err != nil {
								return err
							}
							fmt.Printf("Created version: %s\n", ver.ID)

							if c.Bool("set-latest") {
								update := &model.ModDetails{LatestVersionID: ver.ID}
								if err := repo.UpdateMod(modId, update); err != nil {
									return fmt.Errorf("failed to update latest version: %w", err)
								}
								fmt.Println("Set as latest version.")
							}

							return nil
						},
					},
					{
						Name:      "list",
						Usage:     "List versions for a mod",
						ArgsUsage: "<mod-id>",
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								return fmt.Errorf("mod-id required")
							}
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							ids, err := repo.GetModVersionIds(c.Args().First())
							if err != nil {
								return err
							}
							for _, id := range ids {
								fmt.Println(id)
							}
							return nil
						},
					},

					{
						Name:      "info",
						Usage:     "Get details of a mod version",
						ArgsUsage: "<mod-id> <version-id>",
						Action: func(c *cli.Context) error {
							if c.NArg() < 2 {
								return fmt.Errorf("mod-id and version-id required")
							}
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							mod, err := repo.GetModVersionDetails(c.Args().Get(0), c.Args().Get(1))
							if err != nil {
								return err
							}
							b, err := json.MarshalIndent(mod, "", "  ")
							if err != nil {
								return err
							}
							fmt.Println(string(b))
							return nil
						},
					},
					{
						Name:      "delete",
						Usage:     "Delete a mod version",
						ArgsUsage: "<mod-id> <version-id>",
						Action: func(c *cli.Context) error {
							if c.NArg() < 2 {
								return fmt.Errorf("mod-id and version-id required")
							}
							db, err := gorm.Open(postgres.Open(dbUrl))
							if err != nil {
								return err
							}
							repo := gormrepo.NewGormRepository(db)

							if err := repo.DeleteModVersion(c.Args().Get(0), c.Args().Get(1)); err != nil {
								return err
							}
							fmt.Printf("Deleted version %s from mod %s\n", c.Args().Get(1), c.Args().Get(0))
							return nil
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
